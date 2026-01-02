package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	dbName         = "kapiarso"
	collectionName = "kapiarso"
	serverAddr     = ":5000"
	expiryDays     = 365

	// Rate limiting constants
	maxURLsPerSubnet = 10 // Maksimal 10 shortlink per subnet /24
	cooldownHours    = 24 // Cooldown 24 jam
)

type URLData struct {
	Original      string    `bson:"original_url" json:"original_url"`
	ShortCode     string    `bson:"short_code" json:"short_code"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt     time.Time `bson:"expires_at" json:"expires_at"`
	CreatorIP     string    `bson:"creator_ip,omitempty" json:"creator_ip,omitempty"`
	CreatorSubnet string    `bson:"creator_subnet,omitempty" json:"creator_subnet,omitempty"`
}

type CacheItem struct {
	OriginalURL string
	ExpiresAt   time.Time
}

type RateLimitInfo struct {
	Count     int       // Jumlah URL yang dibuat
	FirstSeen time.Time // Waktu pertama kali membuat URL
	Cooldown  time.Time // Waktu cooldown berakhir (jika melebihi limit)
}

var (
	client *mongo.Client
	col    *mongo.Collection
	cache  = map[string]CacheItem{}

	// Rate limiting berdasarkan subnet
	rateLimitMap   = make(map[string]*RateLimitInfo)
	rateLimitMutex = &sync.RWMutex{}

	// Cleanup untuk rate limit data
	rateLimitCleanupInterval = 1 * time.Hour
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
			log.Println("Warning: .env file not found, using environment variables")
	}

	rand.Seed(time.Now().UnixNano())
	connectMongo()
	initialCleanup()        // Pembersihan pertama kali
	startPeriodicCleanup()  // Pembersihan periodik setiap 1 jam
	startRateLimitCleanup() // Cleanup data rate limiting
	startServer()
}

func connectMongo() {
	uri := os.Getenv("MONGODB_URI")  
	if uri == "" {
			log.Fatal("MONGODB_URI not set in .env file or environment")
	}

	opts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(10 * time.Second)

	c, err := mongo.Connect(opts)
	if err != nil {
		log.Fatal("MongoDB connection failed: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("MongoDB ping failed: ", err)
	}

	client = c
	col = client.Database(dbName).Collection(collectionName)

	log.Println("MongoDB connected successfully")
}

func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	ip := strings.Split(r.RemoteAddr, ":")[0]
	return ip
}

func getClientSubnet(r *http.Request) string {
	ipStr := getClientIP(r)

	// Parse IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		// Jika parsing gagal, return IP asli sebagai fallback
		return ipStr
	}

	// Untuk IPv4: ambil subnet /24 (3 oktet pertama)
	if ipv4 := ip.To4(); ipv4 != nil {
		// Format: xxx.xxx.xxx.0/24
		return fmt.Sprintf("%d.%d.%d.0/24", ipv4[0], ipv4[1], ipv4[2])
	}

	// Untuk IPv6: ambil subnet /64 (8 oktet pertama)
	if ipv6 := ip.To16(); ipv6 != nil {
		// Format: xxxx:xxxx:xxxx:xxxx::/64
		return fmt.Sprintf("%02x%02x:%02x%02x:%02x%02x:%02x%02x::/64",
			ipv6[0], ipv6[1], ipv6[2], ipv6[3],
			ipv6[4], ipv6[5], ipv6[6], ipv6[7])
	}

	// Fallback: return IP asli
	return ipStr
}

func checkRateLimit(subnet string) (bool, *RateLimitInfo, error) {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	now := time.Now()

	// Cek apakah subnet sudah ada di memory cache
	if info, exists := rateLimitMap[subnet]; exists {
		// Reset count jika sudah lewat 24 jam dari first seen
		if now.Sub(info.FirstSeen) >= 24*time.Hour {
			info.Count = 0
			info.FirstSeen = now
			info.Cooldown = time.Time{}
		}

		// Cek apakah dalam cooldown
		if !info.Cooldown.IsZero() && now.Before(info.Cooldown) {
			return false, info, nil
		}

		// Reset cooldown jika sudah lewat
		if !info.Cooldown.IsZero() && now.After(info.Cooldown) {
			info.Cooldown = time.Time{}
		}

		// Cek apakah sudah mencapai limit
		if info.Count >= maxURLsPerSubnet {
			// Set cooldown 24 jam
			info.Cooldown = now.Add(cooldownHours * time.Hour)
			return false, info, nil
		}

		return true, info, nil
	}

	// Subnet tidak ada di memory cache, cek di database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Hitung berapa banyak URL yang dibuat oleh subnet ini dalam 24 jam terakhir
	oneDayAgo := now.Add(-24 * time.Hour)

	count, err := col.CountDocuments(ctx, bson.M{
		"creator_subnet": subnet,
		"created_at":     bson.M{"$gte": oneDayAgo},
	})

	if err != nil {
		return false, nil, err
	}

	// Buat entry baru di memory cache
	rateLimitMap[subnet] = &RateLimitInfo{
		Count:     int(count),
		FirstSeen: now,
		Cooldown:  time.Time{},
	}

	// Cek apakah sudah mencapai limit berdasarkan data database
	if int(count) >= maxURLsPerSubnet {
		rateLimitMap[subnet].Cooldown = now.Add(cooldownHours * time.Hour)
		return false, rateLimitMap[subnet], nil
	}

	return true, rateLimitMap[subnet], nil
}

func incrementRateLimit(subnet string) {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	if info, exists := rateLimitMap[subnet]; exists {
		info.Count++

		// Jika mencapai limit, set cooldown
		if info.Count >= maxURLsPerSubnet {
			info.Cooldown = time.Now().Add(cooldownHours * time.Hour)
		}
	}
}

func startRateLimitCleanup() {
	ticker := time.NewTicker(rateLimitCleanupInterval)

	go func() {
		for {
			<-ticker.C
			cleanupRateLimitMap()
		}
	}()

	log.Println("Rate limit cleanup scheduled (every 1 hour)")
}

func cleanupRateLimitMap() {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	now := time.Now()
	removedCount := 0

	for subnet, info := range rateLimitMap {
		if now.Sub(info.FirstSeen) > 24*time.Hour {
			delete(rateLimitMap, subnet)
			removedCount++
		}
	}

	if removedCount > 0 {
		log.Printf("Rate limit cleanup: removed %d old subnet entries", removedCount)
	}
}

func initialCleanup() {
	log.Println("Running initial cleanup...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()

	result, err := col.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": now},
	})

	if err != nil {
		log.Printf("Error during initial cleanup: %v", err)
		return
	}

	log.Printf("Initial cleanup complete. Deleted %d expired records", result.DeletedCount)

	loadActiveCache()
}

func loadActiveCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	now := time.Now()

	cursor, err := col.Find(ctx, bson.M{
		"expires_at": bson.M{"$gt": now},
	})

	if err != nil {
		log.Printf("Error loading cache: %v", err)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var u URLData
		if cursor.Decode(&u) != nil {
			continue
		}

		cache[u.ShortCode] = CacheItem{
			OriginalURL: u.Original,
			ExpiresAt:   u.ExpiresAt,
		}
	}

	log.Printf("Loaded %d active items to cache", len(cache))
}

func startPeriodicCleanup() {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		for {
			<-ticker.C
			performCleanup()
		}
	}()

	log.Println("Periodic cleanup scheduled (every 1 hour)")
}

func performCleanup() {
	log.Println("Running periodic cleanup...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()

	result, err := col.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": now},
	})

	if err != nil {
		log.Printf("Error during periodic cleanup: %v", err)
		return
	}

	deletedCount := result.DeletedCount

	cleanedCacheCount := 0
	for code, item := range cache {
		if now.After(item.ExpiresAt) {
			delete(cache, code)
			cleanedCacheCount++
		}
	}

	if deletedCount > 0 || cleanedCacheCount > 0 {
		log.Printf("Cleanup complete. Database: %d deleted, Cache: %d cleaned", deletedCount, cleanedCacheCount)
	}
}

func startServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/shorten", shortenHandler)
	mux.HandleFunc("/api/list", listHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/stats", statsHandler)

	mux.HandleFunc("/", mainHandler)

	log.Println("Server running at http://localhost:5000")
	log.Println("Cleanup scheduled every 1 hour")
	log.Printf("Rate limit: %d URLs per subnet (/24) per day", maxURLsPerSubnet)

	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "" {
		http.ServeFile(w, r, "index.html")
		return
	}

	if len(path) == 6 && isAlphanumeric(path) {
		redirectHandler(path, w, r)
		return
	}

	http.NotFound(w, r)
}

func isAlphanumeric(s string) bool {
	if len(s) != 6 {
		return false
	}

	hasLetter := false
	hasDigit := false

	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		} else if c >= '0' && c <= '9' {
			hasDigit = true
		} else {
			return false // Karakter tidak valid
		}
	}

	// Harus memiliki minimal 1 huruf dan 1 angka
	return hasLetter && hasDigit
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	clientSubnet := getClientSubnet(r)

	// Check rate limit berdasarkan subnet
	allowed, rateInfo, err := checkRateLimit(clientSubnet)
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
		http.Error(w, "internal server error", 500)
		return
	}

	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)

		remainingTime := rateInfo.Cooldown.Sub(time.Now())
		json.NewEncoder(w).Encode(map[string]any{
			"success":            false,
			"error":              "rate_limit_exceeded",
			"message":            fmt.Sprintf("Subnet %s telah membuat %d shortlink hari ini. Cooldown %s", clientSubnet, maxURLsPerSubnet, formatDuration(remainingTime)),
			"limit":              maxURLsPerSubnet,
			"subnet":             clientSubnet,
			"client_ip":          clientIP,
			"cooldown_until":     rateInfo.Cooldown.Format(time.RFC3339),
			"cooldown_remaining": formatDuration(remainingTime),
		})
		return
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	if !strings.HasPrefix(body.URL, "http") {
		http.Error(w, "invalid url", 400)
		return
	}

	code := genCode(6)
	now := time.Now()
	exp := now.Add(expiryDays * 24 * time.Hour)

	data := URLData{
		Original:      body.URL,
		ShortCode:     code,
		CreatedAt:     now,
		ExpiresAt:     exp,
		CreatorIP:     clientIP,
		CreatorSubnet: clientSubnet,
	}

	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	_, err = col.InsertOne(ctx, data)

	if err != nil {
		log.Printf("Error inserting to DB: %v", err)
		http.Error(w, "database error", 500)
		return
	}

	cache[code] = CacheItem{body.URL, exp}

	// Increment rate limit counter berdasarkan subnet
	incrementRateLimit(clientSubnet)

	host := r.Host
	if host == "" {
		host = "localhost:5000"
	}
	shortURL := "http://" + host + "/" + code

	// Hitung remaining quota
	remaining := maxURLsPerSubnet - rateInfo.Count - 1 // -1 karena baru saja dibuat
	if remaining < 0 {
		remaining = 0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":      true,
		"short_url":    shortURL,
		"original_url": body.URL,
		"created_at":   now.Format(time.RFC3339),
		"expires_at":   exp.Format(time.RFC3339),
		"expires_in":   formatDuration(exp.Sub(now)),
		"short_code":   code,
		"client_info": map[string]string{
			"ip":     clientIP,
			"subnet": clientSubnet,
		},
		"rate_limit": map[string]any{
			"remaining":      remaining,
			"limit":          maxURLsPerSubnet,
			"reset_in":       formatDuration(24*time.Hour - time.Since(rateInfo.FirstSeen)),
			"current_subnet": clientSubnet,
		},
	})
}

func redirectHandler(code string, w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	if c, ok := cache[code]; ok {
		if now.Before(c.ExpiresAt) {
			http.Redirect(w, r, c.OriginalURL, http.StatusFound)
			return
		} else {
			delete(cache, code)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u URLData
	err := col.FindOne(ctx, bson.M{"short_code": code}).Decode(&u)

	if err == nil {
		if now.Before(u.ExpiresAt) {
			cache[code] = CacheItem{u.Original, u.ExpiresAt}
			http.Redirect(w, r, u.Original, http.StatusFound)
			return
		} else {
			col.DeleteOne(ctx, bson.M{"short_code": code})
		}
	}

	http.NotFound(w, r)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()
	activeItems := make(map[string]CacheItem)

	for code, item := range cache {
		if now.Before(item.ExpiresAt) {
			activeItems[code] = item
		}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"count":       len(activeItems),
		"items":       activeItems,
		"server_time": now.Format(time.RFC3339),
	})
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()
	activeCount := 0
	expiredInCache := 0

	for _, item := range cache {
		if now.Before(item.ExpiresAt) {
			activeCount++
		} else {
			expiredInCache++
		}
	}

	rateLimitMutex.RLock()
	totalSubnets := len(rateLimitMap)
	cooldownSubnets := 0
	for _, info := range rateLimitMap {
		if !info.Cooldown.IsZero() && now.Before(info.Cooldown) {
			cooldownSubnets++
		}
	}
	rateLimitMutex.RUnlock()

	stats := map[string]any{
		"server_time": now.Format(time.RFC3339),
		"cache": map[string]int{
			"total":   len(cache),
			"active":  activeCount,
			"expired": expiredInCache,
		},
		"rate_limit": map[string]any{
			"max_per_subnet":        maxURLsPerSubnet,
			"cooldown_hours":        cooldownHours,
			"total_tracked_subnets": totalSubnets,
			"subnets_in_cooldown":   cooldownSubnets,
			"limit_based_on":        "subnet /24",
		},
		"cleanup_schedule": "every 1 hour",
		"next_cleanup":     now.Add(1 * time.Hour).Format("15:04:05"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	totalCount, err := col.CountDocuments(ctx, bson.M{})
	if err == nil {
		stats["database_total"] = totalCount
	}

	activeDBCount, err := col.CountDocuments(ctx, bson.M{
		"expires_at": bson.M{"$gt": now},
	})
	if err == nil {
		stats["database_active"] = activeDBCount
	}

	// Hitung unique subnets
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$creator_subnet",
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$count": "unique_subnets",
		},
	}

	cursor, err := col.Aggregate(ctx2, pipeline)
	if err == nil {
		defer cursor.Close(ctx2)

		var result []bson.M
		if cursor.All(ctx2, &result) == nil && len(result) > 0 {
			if uniqueSubnets, ok := result[0]["unique_subnets"].(int32); ok {
				stats["unique_creator_subnets"] = int(uniqueSubnets)
			}
		}
	}

	json.NewEncoder(w).Encode(stats)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rateLimitMutex.RLock()
	rateLimitStats := len(rateLimitMap)
	rateLimitMutex.RUnlock()

	json.NewEncoder(w).Encode(map[string]any{
		"status":              "ok",
		"cache_len":           len(cache),
		"rate_limit_subnets":  rateLimitStats,
		"rate_limit_strategy": "per subnet /24",
		"max_per_subnet":      maxURLsPerSubnet,
		"server_time":         time.Now().Format(time.RFC3339),
	})
}

func genCode(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"
	const allChars = letters + digits

	for {
		// Generate random string
		b := make([]byte, n)
		for i := range b {
			b[i] = allChars[rand.Intn(len(allChars))]
		}

		code := string(b)

		// Validasi: harus mengandung minimal 1 huruf dan 1 angka
		if containsLetter(code) && containsDigit(code) {
			return code
		}

		// Jika tidak valid, coba lagi (sangat jarang terjadi untuk n=6)
	}
}

func containsLetter(s string) bool {
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0 seconds"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days %d hours", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
	}
	return fmt.Sprintf("%d seconds", seconds)
}
