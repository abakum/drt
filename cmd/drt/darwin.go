//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ===== 1. Функция отправки сообщения =====
static void sendToSocket(const char* msg, const char* sockPath) {
    int clientSock = socket(AF_UNIX, SOCK_STREAM, 0);
    if (clientSock == -1) return;

    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, sockPath, sizeof(addr.sun_path)-1);

    if (connect(clientSock, (struct sockaddr*)&addr, sizeof(addr)) == -1) {
        close(clientSock);
        return;
    }

    write(clientSock, msg, strlen(msg));
    close(clientSock);
}

// ===== 2. Делегат окна =====
@interface WindowDelegate : NSObject <NSWindowDelegate> {
    const char* sockPath;
}
- (id)initWithSocketPath:(const char*)path;
@end

@implementation WindowDelegate
- (id)initWithSocketPath:(const char*)path {
    self = [super init];
    if (self) {
        sockPath = strdup(path);
    }
    return self;
}

- (void)windowWillClose:(NSNotification *)notification {
    sendToSocket("", sockPath);
}

- (void)dealloc {
    free((void*)sockPath);
    [super dealloc];
}
@end

// ===== 3. View для перетаскивания =====
@interface DragView : NSView <NSDraggingDestination> {
    const char* sockPath;
}
- (id)initWithFrame:(NSRect)frameRect socketPath:(const char*)path;
@end

@implementation DragView
- (id)initWithFrame:(NSRect)frameRect socketPath:(const char*)path {
    self = [super initWithFrame:frameRect];
    if (self) {
        sockPath = strdup(path);
        [self registerForDraggedTypes:@[NSPasteboardTypeFileURL]];
    }
    return self;
}

- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)sender {
    return NSDragOperationCopy;
}

- (BOOL)performDragOperation:(id<NSDraggingInfo>)sender {
    NSPasteboard *pboard = [sender draggingPasteboard];
    NSArray *urls = [pboard readObjectsForClasses:@[[NSURL class]]
                                        options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];

    NSMutableString *combinedPaths = [NSMutableString string];
    for (NSURL *url in urls) {
        if ([combinedPaths length] > 0) [combinedPaths appendString:@"\n"];
        [combinedPaths appendString:[url path]];
    }

    if ([combinedPaths length] > 0) {
        sendToSocket([combinedPaths UTF8String], sockPath);
    }
    return YES;
}

- (void)dealloc {
    free((void*)sockPath);
    [super dealloc];
}
@end

// ===== 4. Главный делегат =====
@interface AppDelegate : NSObject <NSApplicationDelegate> {
    NSString* windowTitle;
    const char* sockPath;
    WindowDelegate* windowDelegate;
}
- (void)setWindowTitle:(NSString*)title;
- (NSString*)windowTitle;
- (void)setSocketPath:(const char*)path;
@end

@implementation AppDelegate
- (void)setWindowTitle:(NSString*)title {
    if (windowTitle) [windowTitle release];
    windowTitle = [title retain];
}

- (NSString*)windowTitle {
    return windowTitle;
}

- (void)setSocketPath:(const char*)path {
    if (sockPath) free((void*)sockPath);
    sockPath = strdup(path);
}

- (void)applicationWillFinishLaunching:(NSNotification *)notification {
    NSWindow *window = [[NSWindow alloc]
        initWithContentRect:NSMakeRect(0, 0, 130, 80)
                  styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
                    backing:NSBackingStoreBuffered
                      defer:NO];

    windowDelegate = [[WindowDelegate alloc] initWithSocketPath:sockPath];
    [window setDelegate:windowDelegate];
    window.level = NSFloatingWindowLevel;
    window.collectionBehavior = NSWindowCollectionBehaviorFullScreenNone;
    window.styleMask &= ~NSWindowStyleMaskResizable;

    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

    DragView *dragView = [[DragView alloc] initWithFrame:window.contentView.bounds socketPath:sockPath];
    [window.contentView addSubview:dragView];
    [dragView release];

    // Исправленная строка - используем метод windowTitle вместо прямого доступа
    [window setTitle:[self windowTitle] ?: @"dr&Tags"];
    [window center];
    [window makeKeyAndOrderFront:nil];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}

- (void)dealloc {
    if (windowTitle) [windowTitle release];
    if (sockPath) free((void*)sockPath);
    [windowDelegate release];
    [super dealloc];
}
@end

// ===== 5. Функция запуска =====
void StartApp(const char* title, const char* sockPath) {
    [NSApplication sharedApplication];
    AppDelegate *delegate = [[AppDelegate alloc] init];

    if (title != NULL) {
        [delegate setWindowTitle:[NSString stringWithUTF8String:title]];
    }

    if (sockPath != NULL) {
        [delegate setSocketPath:sockPath];
    }

    [NSApp setDelegate:delegate];
    [NSApp run];

    [delegate release];
}
*/
import "C"

/*
// ===== 5. Функция запуска =====
void StartApp(const char* title, const char* sockPath) {
    [NSApplication sharedApplication];

    // Важно: Устанавливаем политику ДО создания окон
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];

    AppDelegate *delegate = [[AppDelegate alloc] init];
    [delegate setWindowTitle:[NSString stringWithUTF8String:title ?: ""]];
    [delegate setSocketPath:sockPath];

    [NSApp setDelegate:delegate];
    [NSApp activateIgnoringOtherApps:YES]; // Принудительная активация
    [NSApp run];
}
*/

/*
// ===== 5. Функция запуска =====
void StartApp(const char* title, const char* sockPath) {
    [NSApplication sharedApplication];
    AppDelegate *delegate = [[AppDelegate alloc] init];

    if (title != NULL) {
        [delegate setWindowTitle:[NSString stringWithUTF8String:title]];
    }

    if (sockPath != NULL) {
        [delegate setSocketPath:sockPath];
    }

    [NSApp setDelegate:delegate];
    [NSApp run];

    [delegate release];
}
*/

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"github.com/adrg/xdg"
	"github.com/google/shlex"
	"github.com/xlab/closer"
)

func unixSocketListener(sock string) {
	os.Remove(sock)

	sockFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Println("Socket error:", err)
		return
	}
	defer syscall.Close(sockFd)

	addr := &syscall.SockaddrUnix{Name: sock}
	if err := syscall.Bind(sockFd, addr); err != nil {
		log.Println("Bind error:", err)
		return
	}

	if err := syscall.Listen(sockFd, 5); err != nil {
		log.Println("Listen error:", err)
		return
	}

	// log.Println("UNIX socket listener started on", sock)

	for {
		fd, _, err := syscall.Accept(sockFd)
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		buf := make([]byte, 102400)
		n, err := syscall.Read(fd, buf)
		syscall.Close(fd)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}
		if n == 0 {
			closer.Close()
			return
		}

		paths := string(buf[:n])
		dropPaths(paths)
	}
}

func showDroplet(title string) {
	reg, err := os.CreateTemp("", "drt*.sock")
	if err != nil {
		log.Println("Failed to create temp socket:", err)
		return
	}
	sock := reg.Name()
	// log.Println("Using socket:", sock)
	reg.Close()
	os.Remove(sock)

	go unixSocketListener(sock)

	cTitle := C.CString(title)
	cSock := C.CString(sock)
	closer.Bind(func() {
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cSock))
		os.Remove(sock)
		// выход тут
	})

	C.StartApp(cTitle, cSock)
	// никогда
}

// ------------------
const (
	applescript = "main.applescript"
	Resources   = "Resources"
)

//go:embed drTags.app
var app embed.FS

//go:embed drTags.workflow
var workflow embed.FS

var (
	met = map[string]string{
		dotCSV:  "public.comma-separated-values-text",
		dotMOV:  "com.apple.quicktime-movie",
		dotMP4:  "public.mpeg-4",
		dotFLAC: "org.xiph.flac",
		dotMP3:  "public.mp3",
	}
	DRS = []string{
		filepath.Join(filepath.Dir(xdg.DataHome),
			"Containers",
			"com.blackmagic-design.DaVinciResolveLite",
			"Data",
			"Library",
			"Application Support",
		),
		filepath.Join(xdg.DataHome, "Blackmagic Design", "DaVinci Resolve"),
		filepath.Join(xdg.DataDirs[0], "Blackmagic Design", "DaVinci Resolve"),
	}
	as   = `Display dialog "Установите drTags" buttons {"OK"}'`
	asOk bool
	once bool
)

func onMain() {
	var files []string
	if len(os.Args) > 1 {
		files = os.Args[1:]
	} else {
		finder = getSelectedFiles()
		files = finder[:]
	}
	if strings.ToLower(args0) != droplet {
		return
	}

	// log.Println(files)
	if !once {
		once = true
		MacOS := filepath.Dir(os.Args[0])
		Contents := filepath.Dir(MacOS)
		scriptName := filepath.Join(Contents, Resources, applescript)
		script, err := os.ReadFile(scriptName)
		if err == nil {
			as = string(script)
			asOk = true
		}
	}
	if len(files) == 0 {
		exec.Command("osascript", "-e", "beep").Start()
		cleanup, err := initializeAppLock(drTags)
		if err == nil {
			closer.Bind(cleanup)
			showDroplet(title)
		}
	} else {
		dropPaths(strings.Join(files, "\n"))
	}
	closer.Close()
}

func dropPaths(paths string) {
	files := strings.Split(paths, "\n")
	if len(files) == 0 {
		return
	}
	opts := []string{"-e", as}
	if asOk {
		opts = append(opts, files...)
	}
	cmd := exec.Command("osascript", opts...)
	output, err := cmd.CombinedOutput()
	// log.Println(cmd, err)
	if err != nil {
		log.Println(string(output))
		log.Println(err)
	}
}

// https://github.com/RichardBronosky/AppleScript-droplet
func install(oldname string, lnks ...string) {
	adr, link := lnks[0], lnks[1]
	// /dest/drTags.app dir/drTags
	servicePath := filepath.Join(filepath.Dir(xdg.DataHome), "Services", drTags+".workflow") // dir
	if oldname == "" {
		//uninstall
		log.Println(adr, "~> nul", os.RemoveAll(adr))
		log.Println(servicePath, "~> nul", os.RemoveAll(servicePath))
		log.Println(link, "~> nul", os.Remove(link))
		return
	}
	ln(oldname, link, true, false)

	// Грязный результат fn
	var script []byte

	// Грязные параметры для fn
	_, err := exec.LookPath(drTags)
	replace := err != nil
	dest := filepath.Dir(adr)
	efs := app

	// Walk through the embedded directory and copy files/dirs.
	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, path)
		// log.Println(path, destPath, d.Name(), "path,destPath,d.Name()")

		if d.IsDir() {
			// Create destination directory if it doesn't exist.
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				err = os.MkdirAll(destPath, 0755)
				if err != nil {
					log.Println("Error creating directory:", err)
					return err
				}
			}
			return nil
		}

		if filepath.Base(path) == applescript {
			script, err = efs.ReadFile(path)
			log.Println("<~", path, err)
			if err != nil {
				return err
			}
			if replace {
				script = bytes.Replace(script, []byte(`"`+drTags+`"`), []byte(`"`+filepath.Join(dir, drTags)+`"`), 1)
				err = os.WriteFile(destPath, script, 0644)
				log.Println("~>", path, err)
				if err != nil {
					return err
				}
				return nil
			}
		}

		// Copy file.
		srcFile, err := efs.Open(path)
		if err != nil {
			log.Println("Error opening embedded file:", err)
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			log.Println("Error creating destination file:", err)
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		log.Println(path, "~>", destPath, err)
		if err != nil {
			return err
		}
		return nil
	}

	fs.WalkDir(efs, ".", fn)
	Contents := filepath.Join(adr, "Contents")
	MacOS := filepath.Join(Contents, "MacOS")
	drtlet := filepath.Join(MacOS, droplet)
	os.MkdirAll(MacOS, 0755)
	ln(oldname, drtlet, true, false)
	log.Println("chmod +x", drtlet, os.Chmod(drtlet, 0755))

	dest = filepath.Dir(servicePath)
	efs = workflow
	fs.WalkDir(efs, ".", fn)
	pbs(servicePath, string(script))

}

func evtp() {
}

func SplitCommandLine(command string) ([]string, error) {
	return shlex.Split(command)
}

func mkLink(oldname, newname string, link, hard bool) (err error) {
	return ln(oldname, newname, link, hard)
}

func pbs(servicePath, workflowScript string) {
	documentFile := filepath.Join(servicePath, "Contents", "document.wflow")

	// Чтение файла с логгированием
	documentBytes, err := os.ReadFile(documentFile)
	log.Println("<~", documentFile, err)
	if err != nil {
		return
	}

	documentContent := string(documentBytes)
	// log.Printf(documentContent)
	// log.Printf(workflowScript)

	// Улучшенное регулярное выражение для поиска блока ActionParameters
	re := regexp.MustCompile(`(?s)<key>ActionParameters</key>\s*<dict>\s*<key>source</key>\s*<string>(.*?)</string>`)

	// Заменяем содержимое CDATA
	if re.MatchString(documentContent) {
		// Заменяем существующий скрипт
		documentContent = re.ReplaceAllString(documentContent,
			`<key>ActionParameters</key>
                <dict>
                    <key>source</key>
                    <string><![CDATA[`+workflowScript+`]]></string>`)
	} else {
		// Если не нашли стандартную структуру, вставляем в более общем месте
		re = regexp.MustCompile(`(?s)<key>source</key>\s*<string>(.*?)</string>`)
		if re.MatchString(documentContent) {
			documentContent = re.ReplaceAllString(documentContent,
				`<key>source</key>
                    <string><![CDATA[`+workflowScript+`]]></string>`)
		} else {
			log.Println("Не удалось найти место для вставки скрипта в document.wflow")
			return
		}
	}

	// Запись обратно с логгированием
	err = os.WriteFile(documentFile, []byte(documentContent), 0644)
	log.Println("~>", documentFile, err)
	if err != nil {
		return
	}

	// Обновляем сервисы
	cmd := exec.Command("/System/Library/CoreServices/pbs", "-flush")
	err = cmd.Start()
	log.Println(cmd, err)
	if err != nil {
		return
	}
	cmd.Wait()
}

// osascript  ~/Library/Services/drTags.workflow/Contents/Resources/main.applescript '/Users/koka/Downloads/20200307 Конкурс Варшавской 03 Метнер Две сказки часть 1.mp4'
// tell application "Finder"
//     activate
//     -- Снимаем выделение во всех окнах
//     repeat with win in windows
//         set selection of win to {}
//     end repeat
// end tell
