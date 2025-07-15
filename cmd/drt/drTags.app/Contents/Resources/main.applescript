on runInput(input)
    set shellCmd to "drTags"
    if (count of input) > 0 then
        set quotedPaths to {}
        repeat with anItem in input
            try
                set end of quotedPaths to quoted form of (POSIX path of (anItem as alias))
            end try
        end repeat

        if (count of quotedPaths) > 0 then
            set AppleScript's text item delimiters to " "
            set shellCmd to shellCmd & " " & (quotedPaths as text)
            set AppleScript's text item delimiters to ""
        end if
    end if


    set terminalInstalled to false
 
    try
        tell application "System Events"
            set terminalPath to (path to application "Terminal" as text)
            set terminalInstalled to true
        end tell
    on error
        set terminalInstalled to false
    end try

    if not terminalInstalled then
        set the clipboard to shellCmd
        display dialog "Terminal.app is not installed." &  return & ¬
                    "Command:" & return & shellCmd & return & ¬
                    "in clipboard. Paste it in other Terminal by ⌘V" ¬
                    buttons {"OK"} default button 1 with icon caution
        return
    end if

    set bundleID to "com.github.abakum.drt"
    tell application "Terminal"
        if (exists window 1) then
            set currentTab to front tab of window 1
            set isBusy to busy of currentTab
            if not isBusy  then
                try
                    do script shellCmd in window 1
                on error
                    do script shellCmd
                end try
            else
                do script shellCmd
            end if
        else
            do script shellCmd
        end if
        
        set custom title of first tab of front window to bundleID
        activate
        end tell

end runInput

on run args

    if (class of args is list) and (count of args is 2) then
        set input to item 1 of args
        set parameters to item 2 of args
        
        if class of parameters is record then
            runInput(input)
            return
        end if
    end if

    set aliasList to {}

    if class of args is text then
        try
           set end of aliasList to (POSIX file args) as alias
        end try
    else if class of args is list then
        repeat with itemPath in args
            try
                if class of itemPath is alias then
                    set end of aliasList to itemPath
                else if class of itemPath is text then
                    set end of aliasList to (POSIX file itemPath) as alias
                end if
            end try
        end repeat
    end if 
    runInput(aliasList)

    return aliasList
end run
