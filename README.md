# ⚡ FlashURL - URL Shortener

> **Persingkat URL Anda Secepat Kilat**

FlashURL adalah layanan URL shortener modern yang dibangun dengan Go dan MongoDB, dilengkapi dengan antarmuka frontend yang responsif dengan tema electric yang khas.

## 👥 Tim Pengembang

**Kelas 2IA01 - Kelompok 4**

| Nama | NPM |
|------|------|
| Achmad Fajar | 50424015 |
| Arbaz Ferdiansah | 50424161 |
| Emilio Sinji Sumule | 50424372 |
| Rangga Ho | 51424167 |
| Sultan Ghaniyy Halim | 51424315 |
| Yordan Kapiarso | 51424386 |

## ✨ Features

- ⚡ **Lightning Fast** - Proses shortening dalam milidetik
- 🔒 **Rate Limiting** - Proteksi berbasis subnet /24 untuk mencegah abuse
- 📊 **Real-time Statistics** - Monitoring cache dan database secara live
- 🎨 **Modern UI** - Design dark theme dengan electric accent yang eye-catching
- 🗄️ **MongoDB Integration** - Persistent storage dengan auto-cleanup
- ⏰ **Auto Expiry** - Link otomatis kadaluarsa setelah 365 hari
- 💾 **In-Memory Cache** - Performa optimal dengan sistem caching
- 🛡️ **Security First** - Validasi URL, IP tracking, dan environment variables
- 📱 **Responsive Design** - Mobile-friendly interface
- 🎯 **Clean Code** - Well-structured dan mudah di-maintain

## 🚀 Quick Start

### Prerequisites

- Go 1.21 atau lebih tinggi
- MongoDB Atlas account (atau MongoDB lokal)
- Git

### Installation

1. **Clone repository**
```bash
git clone https://github.com/ArbazFerdiansah/URL_Shortener.git
cd URL_Shortener
```

2. **Install dependencies**
```bash
go mod download
```

3. **Setup Environment Variables**
```bash
cp .env.example .env
```

Edit file `.env` dan isi dengan credentials MongoDB Anda:
```env
MONGODB_URI=mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority
DB_NAME=kapiarso
COLLECTION_NAME=kapiarso
SERVER_PORT=5000
MAX_URLS_PER_SUBNET=10
COOLDOWN_HOURS=24
EXPIRY_DAYS=365
```

4. **Run the server**
```bash
go run main.go
```

Server akan berjalan di `http://localhost:5000`

## 📁 Project Structure

```
URL_Shortener/
├── main.go              # Backend server (Go)
├── go.mod               # Go dependencies
├── go.sum               # Go checksums
├── index.html           # Frontend (All-in-one)
├── .env.example         # Environment template
├── .gitignore           # Git ignore rules
└── README.md            # Documentation
```

## 🔧 Configuration

### Rate Limiting Settings

Default configuration dapat diubah di file `.env`:

```env
MAX_URLS_PER_SUBNET=10    # Maksimal shortlink per subnet /24
COOLDOWN_HOURS=24         # Durasi cooldown (jam)
EXPIRY_DAYS=365           # Masa berlaku link (hari)
```

## 📡 API Documentation

### 1. Shorten URL

**Endpoint:** `POST /api/shorten`

**Request:**
```json
{
  "url": "https://example.com/very-long-url"
}
```

**Response (Success):**
```json
{
  "success": true,
  "short_url": "http://localhost:5000/Rx9zQ4",
  "original_url": "https://example.com/very-long-url",
  "short_code": "Rx9zQ4",
  "created_at": "2025-01-02T15:04:05Z",
  "expires_at": "2026-01-02T15:04:05Z",
  "expires_in": "365 days 0 hours",
  "client_info": {
    "ip": "192.168.1.1",
    "subnet": "192.168.1.0/24"
  },
  "rate_limit": {
    "remaining": 9,
    "limit": 10,
    "reset_in": "23 hours 59 minutes",
    "current_subnet": "192.168.1.0/24"
  }
}
```

**Response (Rate Limited):**
```json
{
  "success": false,
  "error": "rate_limit_exceeded",
  "message": "Subnet 192.168.1.0/24 telah membuat 10 shortlink hari ini. Cooldown 23 hours 45 minutes",
  "limit": 10,
  "subnet": "192.168.1.0/24",
  "client_ip": "192.168.1.1",
  "cooldown_until": "2025-01-03T14:49:00Z",
  "cooldown_remaining": "23 hours 45 minutes"
}
```

### 2. Redirect to Original URL

**Endpoint:** `GET /{short_code}`

**Example:** `GET /Rx9zQ4`

Redirects to original URL with HTTP 302 status.

### 3. List Active URLs

**Endpoint:** `GET /api/list`

**Response:**
```json
{
  "count": 5,
  "items": {
    "Rx9zQ4": {
      "OriginalURL": "https://example.com",
      "ExpiresAt": "2026-01-02T15:04:05Z"
    }
  },
  "server_time": "2025-01-02T15:04:05Z"
}
```

### 4. Statistics

**Endpoint:** `GET /api/stats`

**Response:**
```json
{
  "server_time": "2025-01-02T15:04:05Z",
  "cache": {
    "total": 10,
    "active": 8,
    "expired": 2
  },
  "rate_limit": {
    "max_per_subnet": 10,
    "cooldown_hours": 24,
    "total_tracked_subnets": 5,
    "subnets_in_cooldown": 1,
    "limit_based_on": "subnet /24"
  },
  "database_total": 10,
  "database_active": 8,
  "unique_creator_subnets": 5,
  "cleanup_schedule": "every 1 hour",
  "next_cleanup": "16:04:05"
}
```

### 5. Health Check

**Endpoint:** `GET /api/health`

**Response:**
```json
{
  "status": "ok",
  "cache_len": 10,
  "rate_limit_subnets": 5,
  "rate_limit_strategy": "per subnet /24",
  "max_per_subnet": 10,
  "server_time": "2025-01-02T15:04:05Z"
}
```

## 🎨 Frontend Features

### User Interface
- **Real-time URL validation** - Validasi sebelum submit
- **Visual feedback** - Loading states, error messages, success animations
- **Responsive design** - Optimized untuk mobile dan desktop
- **Copy to clipboard** - One-click copy functionality
- **Particle effects** - Celebratory animations on success
- **Dark theme** - Electric-themed dark mode
- **Smooth animations** - Modern transitions dan micro-interactions

### Technologies Used (Frontend)
- **Tailwind CSS** - Utility-first CSS framework
- **Font Awesome** - Icon library
- **Space Grotesk Font** - Modern typography
- **Vanilla JavaScript** - No framework dependencies

## 🔐 Security Features

- **Rate Limiting** - Berbasis subnet /24 untuk mencegah spam
- **URL Validation** - Validasi format URL sebelum shortening
- **IP Tracking** - Logging IP dan subnet untuk monitoring
- **Auto Cleanup** - Automatic deletion of expired links
- **Environment Variables** - Sensitive data protection
- **Input Sanitization** - Protection against injection attacks

## 🛠️ Development

### Build

```bash
# Build binary
go build -o flashurl main.go

# Run binary
./flashurl
```

### Testing

```bash
# Test shorten endpoint
curl -X POST http://localhost:5000/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://google.com"}'

# Test redirect
curl -L http://localhost:5000/{short_code}

# Test statistics
curl http://localhost:5000/api/stats

# Test health check
curl http://localhost:5000/api/health
```

## 📊 System Architecture

### Backend Components

1. **HTTP Server** - Go native `net/http`
2. **Database** - MongoDB with connection pooling
3. **Cache Layer** - In-memory map for fast lookups
4. **Rate Limiter** - Subnet-based with automatic cleanup
5. **Auto Cleanup** - Scheduled goroutine for expired entries

### Data Flow

```
User Request → Rate Limit Check → URL Validation → MongoDB Insert
                                                           ↓
                                                    Cache Update
                                                           ↓
Redirect ← Cache Lookup ← Short Code ← Response ← Generate Short Code
```


## 📈 Performance Optimization

- **In-memory Caching** - Mengurangi database queries
- **Connection Pooling** - MongoDB connection optimization
- **Automatic Cleanup** - Scheduled cleanup setiap 1 jam
- **Efficient Short Code Generation** - Random alphanumeric dengan validasi
- **Subnet-based Rate Limiting** - Efficient tracking dengan auto-cleanup


## 🤝 Contributing

Contributions, issues, and feature requests are welcome!

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request


## 🙏 Acknowledgments

- **MongoDB Go Driver** - Official MongoDB driver for Go
- **Tailwind CSS** - Utility-first CSS framework
- **Font Awesome** - Icon library
- **godotenv** - Environment variable management
- **Space Grotesk Font** - Typography by Florian Karsten

## 📞 Contact & Support

**Repository:** [https://github.com/ArbazFerdiansah/URL_Shortener](https://github.com/ArbazFerdiansah/URL_Shortener)

Untuk pertanyaan atau support, silakan buka issue di GitHub repository.

⚡ **Made with Lightning Speed by Kelompok 4 - 2IA01** ⚡

**Universitas Gunadarma**

</div>
