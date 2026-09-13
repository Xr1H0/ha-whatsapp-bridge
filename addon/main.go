package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	goqrcode "github.com/skip2/go-qrcode"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	currentQR   string
	currentQRMu sync.RWMutex
)

type SendMessageRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	MediaPath string `json:"media_path,omitempty"`
}

type SendMessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type StatusResponse struct {
	Connected bool   `json:"connected"`
	Paired    bool   `json:"paired"`
	QRPending bool   `json:"qr_pending"`
	Status    string `json:"status"`
}

func getDataPath() string {
	if p := os.Getenv("DATA_PATH"); p != "" {
		return p
	}
	return "/data"
}

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

func setQR(code string) {
	currentQRMu.Lock()
	defer currentQRMu.Unlock()
	currentQR = code
}

func getQR() string {
	currentQRMu.RLock()
	defer currentQRMu.RUnlock()
	return currentQR
}

func sendMessage(client *whatsmeow.Client, recipient, message, mediaPath string) (bool, string) {
	if !client.IsConnected() {
		return false, "not connected to WhatsApp"
	}

	var jid types.JID
	if strings.Contains(recipient, "@") {
		var err error
		jid, err = types.ParseJID(recipient)
		if err != nil {
			return false, fmt.Sprintf("invalid JID: %v", err)
		}
	} else {
		jid = types.JID{User: recipient, Server: "s.whatsapp.net"}
	}

	msg := &waProto.Message{}

	if mediaPath != "" {
		data, err := os.ReadFile(mediaPath)
		if err != nil {
			return false, fmt.Sprintf("cannot read media: %v", err)
		}
		ext := strings.ToLower(filepath.Ext(mediaPath))
		var mediaType whatsmeow.MediaType
		var mime string
		switch ext {
		case ".jpg", ".jpeg":
			mediaType, mime = whatsmeow.MediaImage, "image/jpeg"
		case ".png":
			mediaType, mime = whatsmeow.MediaImage, "image/png"
		case ".mp4":
			mediaType, mime = whatsmeow.MediaVideo, "video/mp4"
		default:
			mediaType, mime = whatsmeow.MediaDocument, "application/octet-stream"
		}
		resp, err := client.Upload(context.Background(), data, mediaType)
		if err != nil {
			return false, fmt.Sprintf("upload failed: %v", err)
		}
		switch mediaType {
		case whatsmeow.MediaImage:
			msg.ImageMessage = &waProto.ImageMessage{
				Caption: proto.String(message), Mimetype: proto.String(mime),
				URL: &resp.URL, DirectPath: &resp.DirectPath, MediaKey: resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256, FileSHA256: resp.FileSHA256, FileLength: &resp.FileLength,
			}
		case whatsmeow.MediaVideo:
			msg.VideoMessage = &waProto.VideoMessage{
				Caption: proto.String(message), Mimetype: proto.String(mime),
				URL: &resp.URL, DirectPath: &resp.DirectPath, MediaKey: resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256, FileSHA256: resp.FileSHA256, FileLength: &resp.FileLength,
			}
		default:
			name := filepath.Base(mediaPath)
			msg.DocumentMessage = &waProto.DocumentMessage{
				Title: proto.String(name), FileName: proto.String(name), Caption: proto.String(message),
				Mimetype: proto.String(mime), URL: &resp.URL, DirectPath: &resp.DirectPath, MediaKey: resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256, FileSHA256: resp.FileSHA256, FileLength: &resp.FileLength,
			}
		}
	} else {
		msg.Conversation = proto.String(message)
	}

	if _, err := client.SendMessage(context.Background(), jid, msg); err != nil {
		return false, fmt.Sprintf("send failed: %v", err)
	}
	return true, fmt.Sprintf("sent to %s", recipient)
}

var qrPageTmpl = template.Must(template.New("qr").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>WhatsApp Bridge — QR Pairing</title>
<meta http-equiv="refresh" content="30">
<style>
body{font-family:sans-serif;display:flex;flex-direction:column;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f0f2f5}
h1{color:#128c7e}img{margin:20px;border:10px solid white;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,.2)}
p{color:#666;max-width:400px;text-align:center}
</style>
</head>
<body>
<h1>WhatsApp Bridge</h1>
{{if .QRAvailable}}
<p>Open WhatsApp on your phone → Linked Devices → Link a Device → scan this code:</p>
<img src="/qr.png" width="256" height="256" alt="QR Code">
<p><small>Page auto-refreshes every 30 seconds. QR code expires after ~60 seconds.</small></p>
{{else if .Connected}}
<p style="color:#128c7e;font-size:1.2em">&#10003; Connected to WhatsApp</p>
<p>The bridge is running and ready to send messages.</p>
{{else}}
<p>Waiting for QR code... Refresh in a few seconds.</p>
{{end}}
</body>
</html>`))

func startServer(client *whatsmeow.Client, port string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Recipient == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ok, msg := sendMessage(client, req.Recipient, req.Message, req.MediaPath)
		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(SendMessageResponse{Success: ok, Message: msg})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		qr := getQR()
		json.NewEncoder(w).Encode(StatusResponse{
			Connected: client.IsConnected(),
			Paired:    client.Store.ID != nil,
			QRPending: qr != "",
			Status:    func() string {
				if client.IsConnected() {
					return "connected"
				}
				if qr != "" {
					return "waiting_for_qr_scan"
				}
				return "disconnected"
			}(),
		})
	})

	mux.HandleFunc("/qr.png", func(w http.ResponseWriter, r *http.Request) {
		qr := getQR()
		if qr == "" {
			http.Error(w, "no QR code available", http.StatusNotFound)
			return
		}
		png, err := goqrcode.Encode(qr, goqrcode.Medium, 256)
		if err != nil {
			http.Error(w, "QR generation failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(png)
	})

	mux.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct {
			QRAvailable bool
			Connected   bool
		}{
			QRAvailable: getQR() != "",
			Connected:   client.IsConnected(),
		}
		qrPageTmpl.Execute(w, data)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/qr", http.StatusFound)
	})

	fmt.Printf("REST API listening on :%s — QR setup at http://<ha-ip>:%s/qr\n", port, port)
	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()
}

func main() {
	dataPath := getDataPath()
	port := getPort()

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		fmt.Printf("Cannot create data dir %s: %v\n", dataPath, err)
		os.Exit(1)
	}

	logger := waLog.Stdout("WhatsApp", "INFO", true)
	dbLog := waLog.Stdout("DB", "WARN", true)

	dbFile := fmt.Sprintf("file:%s/whatsapp.db?_foreign_keys=on", dataPath)
	container, err := sqlstore.New(context.Background(), "sqlite3", dbFile, dbLog)
	if err != nil {
		logger.Errorf("DB init failed: %v", err)
		os.Exit(1)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			deviceStore = container.NewDevice()
		} else {
			logger.Errorf("Device store failed: %v", err)
			os.Exit(1)
		}
	}

	client := whatsmeow.NewClient(deviceStore, logger)
	client.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Connected:
			setQR("")
			logger.Infof("Connected to WhatsApp")
		case *events.LoggedOut:
			logger.Warnf("Logged out — navigate to /qr to re-pair")
		case *events.Message:
			// incoming messages are ignored; we only send
		}
	})

	startServer(client, port)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		if err := client.Connect(); err != nil {
			logger.Errorf("Connect failed: %v", err)
			os.Exit(1)
		}
		fmt.Printf("\nNot paired — navigate to http://<ha-ip>:%s/qr to scan the QR code\n\n", port)
		for evt := range qrChan {
			if evt.Event == "code" {
				setQR(evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Printf("\nOr open: http://<ha-ip>:%s/qr\n", port)
			} else if evt.Event == "success" {
				setQR("")
				fmt.Println("Paired successfully!")
				break
			} else if evt.Event == "timeout" {
				logger.Warnf("QR timeout — refresh /qr for a new code")
			}
		}
	} else {
		if err := client.Connect(); err != nil {
			logger.Errorf("Connect failed: %v", err)
			os.Exit(1)
		}
	}

	time.Sleep(2 * time.Second)
	if client.IsConnected() {
		fmt.Println("WhatsApp bridge ready.")
	} else {
		logger.Warnf("Not connected after startup — check logs")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("Shutting down...")
	client.Disconnect()
}
