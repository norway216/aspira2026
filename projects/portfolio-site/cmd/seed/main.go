package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"portfolio-site/internal/database"
)

// ProjectDef holds all project data in English.
type ProjectDef struct {
	Title           string
	Slug            string
	Summary         string
	Category        string
	TechStack       string
	ContentHTML     string
	SortOrder       int
}

var projects = []ProjectDef{
	{
		Title:     "Iris Recognition System with Android Camera Stream",
		Slug:      "iris-recognition-android-camera",
		Category:  "Medical Imaging",
		TechStack: "C++, OpenCV, ONNX Runtime, FFmpeg, Android, GStreamer",
		Summary:   "An experimental iris recognition system that uses an authorized Android phone camera as a network video source, streaming to a PC or embedded Linux device for eye detection, iris localization, segmentation, normalization, feature extraction, and matching using C++, OpenCV, and ONNX Runtime.",
		ContentHTML: englishIrisAndroid,
	},
	{
		Title:     "Distributed Embedded Device Web Operations Platform",
		Slug:      "distributed-embedded-ops-platform",
		Category:  "Embedded Linux",
		TechStack: "Go, WebSocket, gRPC, SQLite, Prometheus, Grafana, ARM/RK3588",
		Summary:   "A Go-based web operations platform designed for managing fleets of embedded Linux devices (RK3568/RK3588, Jetson, ARM Debian). Features agent-based auto-registration with minimal input, real-time status monitoring, remote command execution, log collection, alerting, and firmware version management across LAN and remote environments.",
		ContentHTML: englishEmbeddedOps,
	},
	{
		Title:     "Distributed Secure Traffic Gateway",
		Slug:      "distributed-secure-traffic-gateway",
		Category:  "Backend Systems",
		TechStack: "Go, TLS, Docker, Kubernetes, Prometheus, Grafana, gRPC",
		Summary:   "A Go-based distributed secure traffic gateway platform supporting multi-node deployment, encrypted communication with TLS and mutual TLS, load balancing, high-concurrency connection handling, user authentication with quota management, node health checking, and a web management dashboard for monitoring traffic and alerts.",
		ContentHTML: englishTrafficGateway,
	},
	{
		Title:     "Embedded Face Detection & Recognition System",
		Slug:      "embedded-face-recognition-system",
		Category:  "Medical Imaging",
		TechStack: "C++20, OpenCV, dlib, Qt/QML, ONNX Runtime",
		Summary:   "A complete embedded face detection and recognition system built with C++20 and Qt/QML. Features real-time camera capture, HOG/CNN face detection, multi-threaded face preprocessing, dlib-based feature extraction and matching, with a clean QML UI showing detection boxes, recognition results, and adjustable parameters.",
		ContentHTML: englishFaceRecognition,
	},
	{
		Title:     "High-Performance Embedded Hardware Wallet",
		Slug:      "embedded-hardware-wallet",
		Category:  "Security",
		TechStack: "C++20, Qt/QML, SQLite, OpenSSL, spdlog, ARM",
		Summary:   "A secure, high-performance hardware wallet designed for embedded system deployment. Features private key generation and encrypted storage, Ed25519 key derivation, transaction signing, distributed backup and recovery, blockchain network verification, and a multi-layered security architecture with OLED/CLI user interface options.",
		ContentHTML: englishHardwareWallet,
	},
	{
		Title:     "LAN Device Discovery & Management Web System",
		Slug:      "lan-device-management-system",
		Category:  "Backend Systems",
		TechStack: "Go, WebSocket, SQLite, Redis, Docker, Prometheus",
		Summary:   "A lightweight Go web system for periodic LAN device discovery and management. Scans the local network for online devices, tracks IP/MAC addresses, monitors online duration and traffic metrics, provides a web dashboard for visualization, and includes user management, alerting, permission control, and audit logging for lab and enterprise environments.",
		ContentHTML: englishLANDevice,
	},
	{
		Title:     "RK3568 Hardware Wallet with Security Enhancements",
		Slug:      "rk3568-hardware-wallet-security",
		Category:  "Security",
		TechStack: "C++20, Qt/QML, SQLite, OpenSSL, spdlog, RK3568",
		Summary:   "An enhanced hardware wallet system targeting the RK3568 platform. Implements a layered Repository-Service-ViewModel-QML architecture with password-based KEK derivation, DEK-encrypted wallet seeds, Ed25519 key pairs, simulated inter-user transfers, and a critical security feature: automatic wallet data deletion after 3 consecutive incorrect password attempts.",
		ContentHTML: englishRK3568Wallet,
	},
	{
		Title:     "Neural Network Iris Recognition System in C++",
		Slug:      "cpp-neural-iris-recognition",
		Category:  "Medical Imaging",
		TechStack: "C++20, OpenCV, ONNX Runtime, Qt/QML, TensorRT, RK3588",
		Summary:   "A production-ready iris recognition system using C++20 with neural network acceleration via ONNX Runtime, OpenVINO, RKNN, and TensorRT. Supports real-time camera capture, iris quality assessment, deep learning-based segmentation, normalization, feature extraction, liveness detection against photo/screen/print attacks, identity registration and matching, with encrypted local template storage for embedded deployment on RK3588 and Jetson platforms.",
		ContentHTML: englishIrisNN,
	},
	{
		Title:     "Multi-Node Traffic Scheduling & Proxy Control Platform",
		Slug:      "traffic-lab-proxy-platform",
		Category:  "Backend Systems",
		TechStack: "Go, TCP/UDP, Docker, Prometheus, Grafana, SQLite",
		Summary:   "An experimental platform for learning and validating network system design through multi-node proxy forwarding, traffic statistics collection, rate limiting with Token Bucket algorithm, load balancing strategies, node heartbeat monitoring with health checking and automatic failover, Prometheus/Grafana visualization, and Docker Compose-based deployment for lab environments.",
		ContentHTML: englishTrafficLab,
	},
}

func main() {
	mdDir := "../.."
	dbPath := "data/portfolio.db"

	if len(os.Args) > 1 {
		mdDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		dbPath = os.Args[2]
	}

	log.Printf("Seeding projects from embedded definitions")
	log.Printf("Database: %s", dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := database.DriverConfig{
		Driver:          "sqlite3",
		DSN:             dbPath,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
	}
	db, err := database.New(cfg, logger)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify we're in the right directory (check for web/templates)
	if _, err := os.Stat("web/templates"); os.IsNotExist(err) {
		// Try the mdDir path for portfolio-site
		altDirs := []string{".", mdDir, filepath.Join(mdDir, "portfolio-site")}
		found := false
		for _, d := range altDirs {
			if info, err := os.Stat(filepath.Join(d, "web", "templates")); err == nil && info.IsDir() {
				if err := os.Chdir(d); err == nil {
					log.Printf("Changed working directory to: %s", d)
					found = true
					break
				}
			}
		}
		if !found {
			log.Printf("WARNING: Could not find web/templates directory; using current dir")
		}
	}

	// Clear existing projects.
	if _, err := db.ExecContext(context.Background(), "DELETE FROM projects"); err != nil {
		log.Fatalf("Failed to clear projects: %v", err)
	}
	log.Printf("Cleared existing projects")

	seeded := 0
	for i, p := range projects {
		p.SortOrder = i // Assign unique sort order per project
		if err := insertProject(db, p); err != nil {
			log.Printf("ERROR: failed to insert %s: %v", p.Title, err)
			continue
		}
		log.Printf("  ✓ %s (gradient %d)", p.Title, (i%9)+1)
		seeded++
	}

	log.Printf("Seeded %d projects successfully (all in English)", seeded)

	// Check for unused variable
	_ = mdDir
}

func insertProject(db *database.DB, p ProjectDef) error {
	query := `INSERT INTO projects (title, slug, summary, description, category, status,
		cover_image, tech_stack, github_url, demo_url, content_markdown, content_html,
		featured, sort_order, published_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'published', ?, ?, '', '', ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	gradientID := (p.SortOrder % 9) + 1
	coverImage := fmt.Sprintf("gradient:%d", gradientID)

	featured := 1
	if p.SortOrder >= 6 {
		featured = 0
	}

	_, err := db.ExecContext(context.Background(), query,
		p.Title, p.Slug, p.Summary, p.Summary, p.Category,
		coverImage, p.TechStack,
		p.ContentHTML, p.ContentHTML,
		featured, p.SortOrder, now, now, now,
	)
	return err
}

// ============================================================
// English-translated architecture document content
// ============================================================

const englishIrisAndroid = `
<h2>1. Project Objective</h2>
<p>This project designs an <strong>experimental iris recognition system</strong> using an authorized Android phone camera as a network video source. The phone streams video over the local network to a PC, Linux host, or embedded device, where a C++ receiver performs eye detection, iris localization, segmentation, normalization, feature extraction, and matching using OpenCV, FFmpeg/GStreamer, and ONNX Runtime.</p>
<blockquote>Note: Standard Android phone cameras are primarily visible-light cameras, suitable for iris recognition algorithm experiments and engineering validation. High-reliability iris recognition typically requires near-infrared (NIR) cameras and NIR illumination.</blockquote>

<h2>2. System Architecture</h2>
<pre><code>Android Phone
    ↓
Camera Capture (Camera2 / NDK Camera)
    ↓
Preprocessing: Auto-focus, exposure, eye ROI cropping
    ↓
Encoding: H.264 / MJPEG
    ↓
Network Streaming: RTSP / HTTP MJPEG / WebSocket
    ↓
PC / RK3588 / Linux C++ Receiver
    ↓
Decode & Extract Frames
    ↓
Face / Eye Detection
    ↓
Iris Localization & Segmentation
    ↓
Normalization (Polar Unwrapping)
    ↓
Feature Extraction (ONNX Runtime)
    ↓
Template Matching & Identity Recognition
</code></pre>

<h2>3. Key Technical Decisions</h2>
<ul>
<li><strong>Camera API:</strong> Android Camera2 API for frame-level control — auto-focus, exposure compensation, and region-of-interest cropping before encoding</li>
<li><strong>Video Encoding:</strong> Hardware H.264 encoding via MediaCodec, or MJPEG for lower latency on constrained networks</li>
<li><strong>Streaming Protocol:</strong> RTSP for low-latency streams (via MediaCodec + RTSP server), or HTTP MJPEG for simple browser-compatible streaming</li>
<li><strong>Receiver Pipeline:</strong> C++ with FFmpeg/GStreamer for decoding, OpenCV for image processing, ONNX Runtime for neural network inference</li>
</ul>

<h2>4. Iris Recognition Pipeline</h2>
<ul>
<li><strong>Eye Detection:</strong> Haar cascades or lightweight CNN for bounding box detection of the eye region</li>
<li><strong>Iris Localization:</strong> Circular Hough transform or segmentation network to find pupil and iris boundaries</li>
<li><strong>Normalization:</strong> Daugman's rubber sheet model to unwrap the iris into a fixed-size rectangular template</li>
<li><strong>Feature Extraction:</strong> ONNX model (e.g., IrisNet or custom CNN) to generate a compact feature vector</li>
<li><strong>Matching:</strong> Hamming distance or cosine similarity between feature vectors for identity verification</li>
</ul>

<h2>5. Deployment Architecture</h2>
<pre><code>[Android Phone] --- WiFi/LAN --- [Linux PC / RK3588]
                                      |
                                 [C++ Application]
                                      |
                    +------------------+------------------+
                    |                                     |
              [Video Decoder]                    [Iris Pipeline]
              (FFmpeg/GStreamer)                 (OpenCV + ONNX)
                    |                                     |
              [Frame Queue]                      [Result Display]
                                                    (Qt / CLI)
</code></pre>

<h2>6. Performance Considerations</h2>
<ul>
<li>Network latency target: &lt;100ms for real-time streaming over WiFi</li>
<li>Frame resolution: 640×480 to 1280×720 for iris detail preservation</li>
<li>ONNX model optimization: INT8 quantization for ARM deployment on RK3588</li>
<li>Multi-threaded pipeline: capture → decode → detect → extract → match, each in separate threads</li>
</ul>
`

const englishEmbeddedOps = `
<h2>1. Design Philosophy</h2>
<p>In embedded development, BSP debugging, edge device management, and lab device operations, common pain points include frequently changing IP addresses, inconsistent system/kernel/BSP/driver versions across devices, the need to monitor CPU, memory, disk, temperature, GPU, USB, WiFi, audio, and camera status in real time, and the overhead of manually entering configuration for every device. Remote devices may also sit behind NAT or firewalls.</p>

<p>The system follows a <strong>minimal input + agent auto-registration + web operations</strong> philosophy:</p>
<pre><code>User provides only IP / initial auth code / device name
        ↓
Device Agent automatically collects full device information
        ↓
Agent registers with the web platform
        ↓
Web dashboard provides full monitoring and remote operations
</code></pre>

<h2>2. Core Capabilities</h2>
<ul>
<li>Device auto-discovery on the LAN with minimal user input</li>
<li>Agent-based auto-registration — collects OS version, kernel version, BSP version, CPU info, memory, disk partitions, temperature sensors, GPU status, USB devices, WiFi, audio, and camera status</li>
<li>Real-time status monitoring dashboard with historical trends</li>
<li>Remote secure command execution — view logs, restart services, adjust audio volume, inspect USB devices</li>
<li>Log collection and centralized search</li>
<li>Alerting — CPU temperature threshold, disk usage, memory pressure, device offline detection</li>
<li>Firmware version management and OTA update orchestration</li>
<li>Role-based access control and full audit logging</li>
</ul>

<h2>3. System Architecture</h2>
<pre><code>┌──────────────────────────────────────────────────┐
│                  Web Dashboard                     │
│         (Go + HTMX / Vue.js + WebSocket)          │
├──────────────────────────────────────────────────┤
│                API & Control Plane                 │
│            (Go HTTP + gRPC + WebSocket)           │
├──────────────────────────────────────────────────┤
│     Device Registry   │   Log Store   │  Auth     │
│     (SQLite/Postgres) │   (SQLite)    │  (JWT)    │
├──────────────────────────────────────────────────┤
│                 Message Bus (NATS / Redis)         │
├──────────────────────────────────────────────────┤
│    Device Agent    Device Agent    Device Agent    │
│    (Go, ~10MB)     (Go, ~10MB)     (Go, ~10MB)    │
│    [RK3588]        [RK3568]        [Jetson]       │
└──────────────────────────────────────────────────┘
</code></pre>

<h2>4. Agent Design</h2>
<p>The lightweight Go agent (&lt;10MB binary) runs on each managed device and is responsible for:</p>
<ul>
<li>Collecting system information on startup: OS, kernel, BSP, CPU, memory, disk, temperatures, GPU, USB topology, WiFi, audio, camera</li>
<li>Registering with the central platform using device fingerprint + auth token</li>
<li>Sending periodic heartbeats with incremental status updates</li>
<li>Listening for and executing remote commands via a secure gRPC or WebSocket channel</li>
<li>Streaming system logs (journald/syslog) to the central log store</li>
<li>Performing OTA updates with rollback support</li>
</ul>

<h2>5. Security Model</h2>
<ul>
<li>TLS encryption for all agent-platform communication</li>
<li>Mutual TLS (mTLS) for device identity verification</li>
<li>JWT-based user authentication with role-based permissions</li>
<li>Command whitelist — only pre-approved commands can be executed remotely</li>
<li>Full audit trail of all operations: who, what, when, which device, result</li>
<li>Agent binary signed and verified before execution</li>
</ul>
`

const englishTrafficGateway = `
<h2>1. Project Objective</h2>
<p>A Go-based distributed secure traffic gateway platform supporting multi-node deployment, encrypted communication, load balancing, high-concurrency connection handling, user management, node monitoring, and traffic statistics. Designed for enterprise internal network security access, lab-authorized network proxying, edge node traffic forwarding, and cross-region service access acceleration.</p>
<blockquote>This document is intended solely for system design in legally authorized network environments.</blockquote>

<h2>2. Core Capabilities</h2>
<ul>
<li>Multi-node distributed deployment with control/data plane separation</li>
<li>TLS encryption with server authentication and optional mutual TLS (mTLS)</li>
<li>User authentication, authorization, quota management, and auditing</li>
<li>High-concurrency TCP/HTTP/WebSocket connection handling</li>
<li>Node health checking, dynamic node registration/removal, and load balancing</li>
<li>Web management dashboard for users, nodes, traffic, and alerts</li>
<li>Prometheus + Grafana monitoring integration</li>
<li>Docker / Kubernetes deployment support</li>
</ul>

<h2>3. Architecture: Control Plane + Data Plane + Management Plane</h2>
<pre><code>┌─────────────────────────────────────────────────────┐
│                   Management Plane                    │
│          Web Dashboard + Admin API (Go)              │
├─────────────────────────────────────────────────────┤
│                   Control Plane                       │
│     Auth Service  │  Node Registry  │  Policy Engine │
│     User Quota    │  Routing Table  │  Cert Manager  │
├─────────────────────────────────────────────────────┤
│                    Data Plane                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ Node A   │  │ Node B   │  │ Node C   │  ...     │
│  │ (Go)     │  │ (Go)     │  │ (Go)     │          │
│  │ TCP/TLS  │  │ TCP/TLS  │  │ TCP/TLS  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
├─────────────────────────────────────────────────────┤
│              Infrastructure & Monitoring             │
│    Docker/K8s  │  Prometheus  │  Grafana  │  Alert  │
└─────────────────────────────────────────────────────┘
</code></pre>

<h2>4. Data Plane Design</h2>
<p>Each data plane node is a Go service that handles the actual traffic forwarding:</p>
<ul>
<li>Accepts incoming TCP/HTTP/WebSocket connections from clients</li>
<li>Applies user-level rate limiting (Token Bucket algorithm)</li>
<li>Forwards traffic to the target destination through encrypted tunnels</li>
<li>Reports real-time connection counts, bytes transferred, and error rates to the control plane</li>
<li>Participates in health checking — responds to control plane heartbeats</li>
<li>Supports graceful draining before removal from the node pool</li>
</ul>

<h2>5. Load Balancing Strategies</h2>
<ul>
<li><strong>Round Robin:</strong> Distributes connections evenly across available nodes</li>
<li><strong>Least Connections:</strong> Routes to the node with the fewest active connections</li>
<li><strong>Weighted:</strong> Assigns traffic proportionally based on node capacity</li>
<li><strong>Geographic:</strong> Routes to the nearest node by latency</li>
</ul>

<h2>6. Security Architecture</h2>
<ul>
<li>All inter-node communication encrypted with TLS 1.3</li>
<li>Certificate-based mutual authentication between control plane and data plane nodes</li>
<li>JWT-based user authentication with short-lived access tokens</li>
<li>Per-user connection quotas and bandwidth limits</li>
<li>Comprehensive audit logging of all authentication and traffic events</li>
</ul>
`

const englishFaceRecognition = `
<h2>1. System Architecture Overview</h2>
<pre><code>┌───────────────────────────────────────────────────────┐
│                    UI Layer (QML)                      │
│  Main Display        │  Parameter Panel                │
│  - Video feed        │  - Threshold adjustment         │
│  - Detection boxes   │  - Model selection              │
│  - Recognition results│  - FPS display                  │
└───────────────────────────────────────────────────────┘
           ↑                         ↑
┌───────────────────────────────────────────────────────┐
│              Application Logic Layer (C++20)           │
│  CameraManager / VideoManager                          │
│  - Camera detection, switching, video file loading    │
│  FaceDetectionManager (OpenCV + dlib)                  │
│  - Face detection (HOG/CNN)                            │
│  - Multi-threaded face cropping & preprocessing        │
│  FaceRecognitionManager (dlib)                         │
│  - Feature extraction / face matching                  │
│  - Multi-threaded batch processing                     │
│  DataPipeline / ThreadPool                             │
│  - CameraThread → PreprocessThread → AIInferenceThread │
│    → RenderThread                                       │
└───────────────────────────────────────────────────────┘
</code></pre>

<h2>2. Face Detection Pipeline</h2>
<ul>
<li><strong>HOG + Linear SVM:</strong> dlib's HOG-based detector — CPU-efficient, good for frontal faces</li>
<li><strong>CNN Detector:</strong> dlib's MMOD CNN detector — higher accuracy, GPU-accelerated via OpenCV DNN</li>
<li><strong>Multi-threaded Processing:</strong> Each camera frame is dispatched to a worker thread for detection, cropping, and preprocessing</li>
<li><strong>Face Alignment:</strong> 68-point facial landmark detection for affine alignment before recognition</li>
</ul>

<h2>3. Face Recognition Pipeline</h2>
<ul>
<li><strong>Feature Extraction:</strong> dlib's ResNet-34 based face recognition model generates a 128-dimensional embedding vector</li>
<li><strong>Face Matching:</strong> Euclidean distance between embeddings; faces with distance below threshold are considered the same identity</li>
<li><strong>Batch Processing:</strong> Unknown faces are batched and processed in parallel for efficient recognition</li>
<li><strong>Database Integration:</strong> Known face embeddings stored in SQLite with identity metadata</li>
</ul>

<h2>4. Threading Architecture</h2>
<pre><code>CameraThread ──→ PreprocessThread[] ──→ InferenceThread[] ──→ RenderThread
     │                    │                      │                  │
  Frame capture     Face detection          Recognition         QML UI
  30/60 FPS         Face alignment         Feature extraction   Overlay boxes
                    Cropping & resize      DB lookup            FPS counter
</code></pre>

<h2>5. QML UI Design</h2>
<ul>
<li>Main video display with real-time detection bounding box overlay</li>
<li>Recognized identity labels rendered above each face</li>
<li>Side panel for threshold adjustment, model selection, and FPS display</li>
<li>Database management view for enrolling new faces and reviewing known identities</li>
</ul>

<h2>6. Performance Targets</h2>
<ul>
<li>Detection: &lt;30ms per frame (HOG), &lt;100ms per frame (CNN) on RK3588</li>
<li>Recognition: &lt;50ms per face for feature extraction and matching</li>
<li>End-to-end pipeline latency: &lt;150ms from capture to rendered result</li>
<li>Supports up to 10 simultaneous faces in a single frame</li>
</ul>
`

const englishHardwareWallet = `
<h2>1. System Objective</h2>
<p>A high-performance, secure hardware wallet suitable for embedded system deployment. Features distributed backup and recovery, network verification capabilities for secure communication with blockchains or remote servers, implemented in C++20 with multi-threading and performance optimization, minimizing external dependencies for independent operation and security.</p>

<h2>2. System Architecture</h2>
<pre><code>┌───────────────────────────────────────────────────────┐
│                 User Interface Layer                    │
│  OLED / Touch Display / CLI                             │
│  - Transaction input & signing                          │
│  - Address display                                      │
│  - Status & error messages                              │
└───────────────────────────────────────────────────────┘
           ↑                         ↑
┌───────────────────────────────────────────────────────┐
│            Application Logic Layer (C++20)              │
│  WalletCore                                             │
│  - Private key management (generation, storage,        │
│    encryption, access control)                          │
│  - Wallet account management                            │
│  TransactionManager                                     │
│  - Transaction construction & signing                   │
│  - Multi-signature support                              │
│  NetworkValidator                                       │
│  - Blockchain communication & verification              │
│  - Remote server authentication                         │
│  BackupManager                                          │
│  - Distributed backup (Shamir's Secret Sharing)         │
│  - Encrypted recovery phrases                           │
└───────────────────────────────────────────────────────┘
           ↑                         ↑
┌───────────────────────────────────────────────────────┐
│                 Security Layer                          │
│  - Hardware RNG for key generation                     │
│  - Secure element integration                           │
│  - Memory locking (mlock) to prevent swapping          │
│  - Side-channel attack mitigations                     │
│  - Encrypted storage with AES-256-GCM                  │
└───────────────────────────────────────────────────────┘
</code></pre>

<h2>3. Key Management Architecture</h2>
<ul>
<li><strong>Key Generation:</strong> Ed25519 key pairs generated from hardware RNG or secure element entropy</li>
<li><strong>Key Derivation:</strong> BIP32 hierarchical deterministic (HD) wallet structure</li>
<li><strong>Key Encryption:</strong> AES-256-GCM with key derived from user passphrase via Argon2id</li>
<li><strong>Key Storage:</strong> Encrypted keys stored in SQLite database with memory-safe access patterns</li>
<li><strong>Backup:</strong> Shamir's Secret Sharing splits the master seed into N-of-M shards</li>
</ul>

<h2>4. Transaction Flow</h2>
<pre><code>User enters transaction details (OLED/CLI)
        ↓
TransactionManager constructs unsigned transaction
        ↓
User reviews and confirms on secure display
        ↓
WalletCore signs with derived Ed25519 private key
        ↓
Signed transaction broadcast via NetworkValidator
        ↓
Confirmation received and displayed to user
</code></pre>

<h2>5. Security Properties</h2>
<ul>
<li>Private keys never leave the device in plaintext</li>
<li>All cryptographic operations performed in memory-locked pages (mlock/mlockall)</li>
<li>Constant-time comparison for PIN/passphrase verification</li>
<li>Automatic wallet lock after configurable idle timeout</li>
<li>Tamper-evident logging of all security-critical operations</li>
<li>Secure boot verification of firmware integrity</li>
</ul>
`

const englishLANDevice = `
<h2>1. System Positioning</h2>
<p>A lightweight <strong>LAN asset discovery + online status monitoring + traffic visualization + web operations management platform</strong>. This is not an intrusion tool or attack system — it is an asset management system designed for corporate internal networks, laboratories, embedded devices, development boards, and test equipment.</p>

<p>Suitable scenarios:</p>
<ul>
<li>Corporate internal LAN device management</li>
<li>Embedded development board online status monitoring</li>
<li>Medical device, ultrasound device, industrial control device status observation</li>
<li>Test lab equipment asset management</li>
<li>Unknown device discovery on local networks</li>
<li>Network traffic anomaly device ranking</li>
<li>Device online duration statistics</li>
<li>Lightweight NMS (Network Management System)</li>
</ul>

<h2>2. Core Features</h2>
<ul>
<li><strong>Online Device Scanning:</strong> Periodic ARP + ICMP + mDNS scanning of the local subnet with configurable intervals</li>
<li><strong>Device Fingerprinting:</strong> MAC vendor lookup, OS detection via TCP/IP stack fingerprinting, open port scanning</li>
<li><strong>Traffic Monitoring:</strong> Per-device traffic rate estimation via SNMP or netstat polling</li>
<li><strong>Online Duration Tracking:</strong> First seen, last seen, and cumulative online time per device</li>
<li><strong>Web Dashboard:</strong> Real-time device table with sorting, filtering, and search</li>
<li><strong>Alerting:</strong> New device detection, device offline, traffic threshold exceeded</li>
<li><strong>User Management:</strong> Role-based access (admin, operator, viewer)</li>
<li><strong>Audit Logging:</strong> All user actions and system events logged with timestamps</li>
</ul>

<h2>3. System Architecture</h2>
<pre><code>┌──────────────────────────────────────────────┐
│              Web Dashboard (Go + HTML/JS)      │
│         Device Table │ Charts │ Alerts         │
├──────────────────────────────────────────────┤
│              REST API + WebSocket              │
│         (Go net/http + gorilla/websocket)      │
├──────────────────────────────────────────────┤
│    Device Store    │   User Store  │  Audit   │
│    (SQLite/Redis)  │   (SQLite)    │  (SQLite) │
├──────────────────────────────────────────────┤
│              Scan Engine (Go)                  │
│   ARP Scanner  │  ICMP Ping  │  mDNS Query   │
│   MAC Lookup   │  Port Scan  │  SNMP Poll    │
├──────────────────────────────────────────────┤
│              Network Interface                 │
│         (raw socket / pcap / gopacket)        │
└──────────────────────────────────────────────┘
</code></pre>

<h2>4. Scan Engine Design</h2>
<ul>
<li><strong>ARP Scanning:</strong> Send ARP requests to all IPs in the subnet; responding devices are online</li>
<li><strong>ICMP Ping:</strong> Fallback for devices that don't respond to ARP</li>
<li><strong>mDNS Discovery:</strong> Listen for mDNS/Bonjour announcements from embedded devices</li>
<li><strong>MAC OUI Lookup:</strong> Resolve vendor from MAC address using IEEE OUI database</li>
<li><strong>Periodic Scanning:</strong> Configurable interval (default 60s) with adaptive rate limiting to avoid network flooding</li>
</ul>

<h2>5. Security Considerations</h2>
<ul>
<li>All scanning is passive/read-only — no packet injection or modification</li>
<li>Web UI protected by JWT authentication with CSRF protection</li>
<li>Role-based access: admin can configure scan parameters, operators can view, viewers have read-only access</li>
<li>Audit logging of all configuration changes and user actions</li>
<li>Network scanning limited to configured subnet ranges only</li>
</ul>
`

const englishRK3568Wallet = `
<h2>1. System Objective</h2>
<p>An extension of the embedded hardware wallet architecture specifically targeting the <strong>RK3568 platform</strong> with enhanced security features. Built with C++20, Qt/QML for the embedded UI, SQLite for data storage, OpenSSL for cryptographic operations, and spdlog for logging, following a clean Repository → Service → ViewModel → QML layered architecture.</p>

<p>Core enhancement: <strong>After 3 consecutive incorrect password attempts, securely delete the current user's wallet data.</strong></p>

<h2>2. Core Features</h2>
<ul>
<li>User registration, login, and logout with secure session management</li>
<li>Password-based KEK (Key Encryption Key) derivation using Argon2id</li>
<li>KEK decrypts DEK (Data Encryption Key)</li>
<li>DEK decrypts wallet seed (BIP39 mnemonic)</li>
<li>Wallet seed derives Ed25519 key pairs via BIP32</li>
<li>Simulated inter-user transfers with transaction signing</li>
<li>Transaction history with SQLite persistence</li>
<li>Audit logging with tamper-evident storage</li>
</ul>

<h2>3. Architecture Layers</h2>
<pre><code>┌──────────────────────────────────────────┐
│            QML UI Layer                   │
│  Login Screen │ Wallet Dashboard          │
│  Transaction Form │ Settings Panel        │
├──────────────────────────────────────────┤
│          ViewModel Layer (C++)            │
│  LoginVM │ WalletVM │ TransactionVM       │
│  SettingsVM │ AuditLogVM                  │
├──────────────────────────────────────────┤
│           Service Layer (C++)             │
│  AuthService │ WalletService              │
│  TransactionService │ AuditService        │
│  CryptoService (OpenSSL wrapper)          │
├──────────────────────────────────────────┤
│         Repository Layer (C++)           │
│  UserRepo │ WalletRepo │ TransactionRepo  │
│  AuditRepo (SQLite via SQLiteCpp)         │
├──────────────────────────────────────────┤
│         Infrastructure                    │
│  SQLite │ OpenSSL 3.x │ spdlog │ Qt 6    │
│  RK3568 (ARM Cortex-A55 × 4)             │
└──────────────────────────────────────────┘
</code></pre>

<h2>4. Password Security Flow</h2>
<pre><code>User enters password
        ↓
Argon2id(password, salt, iterations, memory) → KEK
        ↓
AES-256-GCM Decrypt(KEK, encrypted_DEK) → DEK
        ↓
AES-256-GCM Decrypt(DEK, encrypted_seed) → BIP39 Seed
        ↓
BIP32 Derive(seed, path) → Ed25519 Key Pair
        ↓
Sign transaction / derive addresses
</code></pre>

<h2>5. Anti-Brute-Force Protection</h2>
<ul>
<li>Failed attempt counter stored in SQLite with atomic increment</li>
<li>Counter persisted across application restarts</li>
<li>After 3 consecutive failures: DELETE wallet data for current user from all tables</li>
<li>Rate limiting: exponential backoff between attempts (1s, 2s, 4s)</li>
<li>Audit event logged for every failed attempt and for wallet deletion</li>
<li>UI displays remaining attempts before data loss</li>
</ul>

<h2>6. Cryptographic Details</h2>
<ul>
<li><strong>Argon2id Parameters:</strong> t=3, m=65536 KiB, p=4, salt=32 bytes random</li>
<li><strong>KEK:</strong> 256-bit key derived from password + salt</li>
<li><strong>DEK:</strong> 256-bit randomly generated, encrypted with KEK using AES-256-GCM</li>
<li><strong>Seed Encryption:</strong> AES-256-GCM with DEK, authentication tag verified before use</li>
<li><strong>Ed25519:</strong> Deterministic derivation from BIP32 seed, nonce via RFC 6979</li>
</ul>
`

const englishIrisNN = `
<h2>1. Project Objective</h2>
<p>A production-ready iris recognition system using <strong>C++20</strong> as the core development language, OpenCV for image acquisition and preprocessing, and neural networks via ONNX Runtime, OpenVINO, RKNN, and TensorRT for iris segmentation, feature extraction, and liveness detection. The system supports user registration, iris template generation, identity recognition, and secure template storage, deployable on both Ubuntu PCs and embedded devices (RK3568/RK3588/Jetson).</p>

<h2>2. Core Capabilities</h2>
<ul>
<li>Real-time camera capture of ocular images with quality feedback</li>
<li>Offline recognition from image and video files</li>
<li>Automatic iris region detection and segmentation (neural network-based)</li>
<li>Pupil and iris boundary localization</li>
<li>Iris normalization via polar unwrapping (Daugman's rubber sheet model)</li>
<li>Neural network feature extraction (128–512 dimensional embeddings)</li>
<li>Iris template enrollment and identity recognition (1:1 verification, 1:N identification)</li>
<li>Liveness detection to mitigate photo, screen replay, and print attacks</li>
<li>Encrypted local storage of iris templates (AES-256-GCM)</li>
<li>Embedded hardware acceleration via RKNN (RK3588 NPU) and TensorRT (Jetson GPU)</li>
</ul>

<h2>3. System Pipeline</h2>
<pre><code>Camera Capture (30 FPS, NIR or Visible)
        ↓
Quality Assessment (focus, occlusion, gaze angle, contrast)
        ↓
Iris Segmentation (UNet / DeepLabV3+ via ONNX)
        ↓
Boundary Localization (pupil circle + iris circle)
        ↓
Normalization (polar unwrap, 512×64 template)
        ↓
Feature Extraction (ResNet / EfficientNet backbone)
        ↓
┌───────────────┴───────────────┐
↓                               ↓
Enrollment                    Recognition
- Extract template             - Extract query features
- Encrypt with AES-256-GCM     - Compare with DB templates
- Store in SQLite              - Hamming distance / cosine similarity
- Register identity            - Return best match or reject
</code></pre>

<h2>4. Neural Network Models</h2>
<ul>
<li><strong>Segmentation:</strong> UNet with MobileNetV2 backbone, trained on CASIA/NICE datasets, exported to ONNX with FP16 quantization</li>
<li><strong>Feature Extraction:</strong> ResNet-50 or EfficientNet-B3 fine-tuned with triplet loss, outputting 256-D normalized embeddings</li>
<li><strong>Liveness Detection:</strong> Binary classifier (MobileNetV3) trained on genuine vs. spoof images (photo, screen, print, contact lens)</li>
<li><strong>Inference Engines:</strong> ONNX Runtime (CPU/GPU), OpenVINO (Intel), RKNN (RK3588 NPU), TensorRT (Jetson GPU)</li>
</ul>

<h2>5. Embedded Deployment</h2>
<ul>
<li><strong>RK3588:</strong> 3-core NPU (6 TOPS) via RKNN API for segmentation and feature extraction</li>
<li><strong>Jetson Orin:</strong> TensorRT with FP16 for all three models in parallel</li>
<li><strong>Memory Budget:</strong> &lt;512 MB RAM total for all models and runtime</li>
<li><strong>Inference Latency:</strong> &lt;50ms segmentation + &lt;30ms feature extraction on RK3588 NPU</li>
<li><strong>Storage:</strong> Encrypted SQLite database for iris templates, &lt;1KB per template</li>
</ul>

<h2>6. Security</h2>
<ul>
<li>Iris templates encrypted at rest with AES-256-GCM, key derived from system HSM or TPM</li>
<li>Templates never leave the device in plaintext</li>
<li>Memory-zeroing after template comparison to prevent forensic recovery</li>
<li>Anti-tampering: signed firmware + verified model checksums</li>
<li>All recognition operations logged in audit trail</li>
</ul>
`

const englishTrafficLab = `
<h2>1. Project Objective</h2>
<p>This is not a commercial proxy system — it is an <strong>experimental platform for learning and validating network system design</strong>. The core objective is to master the engineering implementation of high-performance network services through multi-node proxy forwarding, traffic statistics, rate limiting, load scheduling, and monitoring visualization.</p>

<h2>2. Core Learning Objectives</h2>
<ul>
<li>Understand TCP/UDP traffic forwarding principles at the system-call level</li>
<li>Master multi-node proxy architecture design patterns</li>
<li>Implement per-user and per-node traffic statistics collection</li>
<li>Implement rate-limiting strategies (Token Bucket algorithm)</li>
<li>Implement node load balancing strategies (round-robin, least-conn, weighted)</li>
<li>Implement node heartbeat, health checking, and automatic failover</li>
<li>Use Prometheus + Grafana for monitoring visualization</li>
<li>Use Docker Compose to build reproducible lab environments</li>
<li>Build foundations for future study of high-performance C++ network programming, distributed gateways, and edge network services</li>
</ul>

<h2>3. System Architecture</h2>
<pre><code>┌────────────────────────────────────────────────────┐
│              Web Management Dashboard                │
│    (Go HTTP + WebSocket → Vue.js / HTMX frontend)   │
├────────────────────────────────────────────────────┤
│              Control Plane (Go)                      │
│  User Manager │ Node Registry │ Policy Engine       │
│  Traffic Stats │ Rate Limiter │ Alert Manager       │
├────────────────────────────────────────────────────┤
│              Data Plane Nodes (Go)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │ Node 1   │  │ Node 2   │  │ Node 3   │  ...    │
│  │ TCP/UDP  │  │ TCP/UDP  │  │ TCP/UDP  │         │
│  │ Forwarder│  │ Forwarder│  │ Forwarder│         │
│  └──────────┘  └──────────┘  └──────────┘         │
├────────────────────────────────────────────────────┤
│          Monitoring & Infrastructure                │
│  Prometheus │ Grafana │ Docker Compose │ SQLite    │
└────────────────────────────────────────────────────┘
</code></pre>

<h2>4. Rate Limiting: Token Bucket</h2>
<pre><code>┌─────────────────────────────────────┐
│          Token Bucket                │
│  ┌─────────────────────────────┐    │
│  │  ○ ○ ○ ○ ○ ○ ○ ○ ○ ○ ○ ○  │    │  ← Tokens added at rate R
│  │  (max N tokens)             │    │
│  └─────────────────────────────┘    │
│              ↓                       │
│     Each packet/connection           │
│     consumes 1 token                 │
│     If bucket empty → rate limited   │
└─────────────────────────────────────┘
</code></pre>
<ul>
<li><strong>Rate:</strong> Configurable tokens/second per user and per node</li>
<li><strong>Burst:</strong> Maximum bucket size for short bursts above steady rate</li>
<li><strong>Enforcement:</strong> TCP connections delayed or rejected when bucket is empty</li>
</ul>

<h2>5. Node Health & Failover</h2>
<ul>
<li>Nodes send heartbeat to control plane every 5 seconds</li>
<li>Control plane marks node as unhealthy after 3 missed heartbeats (15s timeout)</li>
<li>Unhealthy nodes are removed from the active pool; traffic is redistributed</li>
<li>When a node recovers and resumes heartbeats, it is automatically re-added</li>
<li>Graceful drain: node signals intent to leave, stops accepting new connections, completes existing ones, then disconnects</li>
</ul>

<h2>6. What This Project Does NOT Study</h2>
<ul>
<li>Traffic obfuscation, censorship circumvention, or protocol mimicry</li>
<li>Detection avoidance or anti-probing techniques</li>
<li>Commercial proxy service architectures</li>
<li>Any technique that bypasses network regulations or authorized access controls</li>
</ul>
`
