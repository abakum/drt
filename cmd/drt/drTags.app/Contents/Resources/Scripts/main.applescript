-- cd /Applications/drTags.app/Contents/Resources/Scripts/
-- osadecompile main.scpt > main.applescript
-- osacompile -l AppleScript -d main.scpt > main.applescript
-- osacompile -o main.scpt main.applescript

--https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/ProcessDroppedFilesandFolders.html#//apple_ref/doc/uid/TP40016239-CH53-SW1
property theExtensionsToProcess : {"csv", "mov", "mp4", "flac", "mp3"}
property theTypeIdentifiersToProcess : {
    "public.comma-separated-values-text",
    "com.apple.quicktime-movie",
    "public.mpeg-4",
    "org.xiph.flac",
    "public.mp3",
    "public.folder"
}

on run
    process({})
end run

on open dropped_files
    process(dropped_files)
end open

on process(dropped_files)
    set validFiles to {}
    
    if (count of dropped_files) > 0 then
        repeat with aFile in dropped_files
            tell application "System Events"
                set fileExtension to name extension of aFile
                set fileType to type identifier of aFile
                set filePath to POSIX path of aFile
            end tell
            
            if (theExtensionsToProcess contains fileExtension) or (theTypeIdentifiersToProcess contains fileType) then
                set end of validFiles to quoted form of filePath
            end if
        end repeat
    end if
    
    set shellCmd to "drTags"
    if (count of validFiles) > 0 then
        set shellCmd to shellCmd & " " & joinList(validFiles, " ")
    end if
    
    tell application "Terminal"
        if not (exists window 1) then reopen
        activate
        do script shellCmd in window 1
    end tell
end process

on joinList(theList, delimiter)
    set AppleScript's text item delimiters to delimiter
    set resultString to theList as text
    set AppleScript's text item delimiters to ""
    return resultString
end joinList