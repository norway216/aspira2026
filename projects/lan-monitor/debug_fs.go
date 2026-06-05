package main

import (
    "fmt"
    "io/fs"
)

func listFS() {
    fmt.Println("Walking StaticFS:")
    fs.WalkDir(StaticFS, ".", func(path string, d fs.DirEntry, err error) error {
        if err != nil { 
            fmt.Printf("  ERROR: %v\n", err)
            return nil
        }
        info, _ := d.Info()
        fmt.Printf("  %s (dir=%v, size=%d)\n", path, d.IsDir(), info.Size())
        return nil
    })
    
    fmt.Println("\nReading css/app.css:")
    data, err := fs.ReadFile(StaticFS, "css/app.css")
    if err != nil {
        fmt.Printf("  ERROR: %v\n", err)
    } else {
        fmt.Printf("  OK: %d bytes\n  First line: %s\n", len(data), string(data[:min(50, len(data))]))
    }
    
    fmt.Println("\nReading js/app.js:")
    data2, err := fs.ReadFile(StaticFS, "js/app.js")
    if err != nil {
        fmt.Printf("  ERROR: %v\n", err)
    } else {
        fmt.Printf("  OK: %d bytes\n  First line: %s\n", len(data2), string(data2[:min(50, len(data2))]))
    }
}

func min(a, b int) int { if a < b { return a }; return b }
