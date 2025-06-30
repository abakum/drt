package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const (
	dropletName      = "drt"
	bundleID         = "com.github.abakum." + dropletName
	executableName   = dropletName + "Launcher"
	executableScript = `#!/bin/bash

# Log file location
LOG_FILE=~/Library/Logs/{{.DropletName}}.log

# Process each dropped file
for file in "$@"
do
    echo "$(date): Processing $file" >> "$LOG_FILE"
    # Add your processing logic here
    open -a TextEdit "$file" # Example: open in TextEdit
done
`
	infoPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>{{.ExecutableName}}</string>
    <key>CFBundleIdentifier</key>
    <string>{{.BundleID}}</string>
    <key>CFBundleName</key>
    <string>{{.DropletName}}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>CFBundleDocumentTypes</key>
    <array>
        <dict>
            <key>CFBundleTypeName</key>
            <string>All Files</string>
            <key>CFBundleTypeRole</key>
            <string>Viewer</string>
            <key>LSHandlerRank</key>
            <string>Alternate</string>
            <key>LSItemContentTypes</key>
            <array>
                <string>public.item</string>
            </array>
        </dict>
    </array>
</dict>
</plist>
`
)

type templateData struct {
	DropletName    string
	BundleID       string
	ExecutableName string
}

func main() {
	// Create .app bundle structure
	appPath := filepath.Join("/Applications", dropletName+".app")
	contentsPath := filepath.Join(appPath, "Contents")
	macosPath := filepath.Join(contentsPath, "MacOS")
	resourcesPath := filepath.Join(contentsPath, "Resources")

	if err := os.MkdirAll(macosPath, 0755); err != nil {
		fmt.Printf("Error creating MacOS directory: %v\n", err)
		return
	}
	if err := os.MkdirAll(resourcesPath, 0755); err != nil {
		fmt.Printf("Error creating Resources directory: %v\n", err)
		return
	}

	// Create executable script
	executablePath := filepath.Join(macosPath, executableName)
	data := templateData{
		DropletName:    dropletName,
		BundleID:       bundleID,
		ExecutableName: executableName,
	}

	// Render and write executable script
	tmpl := template.Must(template.New("script").Parse(executableScript))
	execFile, err := os.Create(executablePath)
	if err != nil {
		fmt.Printf("Error creating executable: %v\n", err)
		return
	}
	defer execFile.Close()

	if err := tmpl.Execute(execFile, data); err != nil {
		fmt.Printf("Error writing executable: %v\n", err)
		return
	}

	// Make executable
	if err := os.Chmod(executablePath, 0755); err != nil {
		fmt.Printf("Error making executable: %v\n", err)
		return
	}

	// Create Info.plist
	infoPlistPath := filepath.Join(contentsPath, "Info.plist")
	tmpl = template.Must(template.New("plist").Parse(infoPlistTemplate))
	plistFile, err := os.Create(infoPlistPath)
	if err != nil {
		fmt.Printf("Error creating Info.plist: %v\n", err)
		return
	}
	defer plistFile.Close()

	if err := tmpl.Execute(plistFile, data); err != nil {
		fmt.Printf("Error writing Info.plist: %v\n", err)
		return
	}

	fmt.Printf("Successfully created %s.app\n", dropletName)
}
