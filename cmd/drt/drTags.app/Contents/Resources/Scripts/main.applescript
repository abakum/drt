-- cd /Applications/drTags.app/Contents/Resources/Scripts/
-- osadecompile main.scpt > main.applescript
-- osacompile -l AppleScript -d main.scpt > main.applescript
-- osacompile -o main.scpt main.applescript

--https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/ProcessDroppedFilesandFolders.html#//apple_ref/doc/uid/TP40016239-CH53-SW1
property theExtensionsToProcess : {"csv", "mov", "mp4", "flac", "mp3"}
property theTypeIdentifiersToProcess : {"public.comma-separated-values-text", "com.apple.quicktime-movie", "public.mpeg-4", "org.xiph.flac", "public.mp3", "public.folder"}

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

	if not (application "Terminal" exists) then
		set commandToRun to "open -a drTags.app"
		set the clipboard to commandToRun
		display dialog "Paste" & commandToRun & "by press Cmd+V" buttons {"OK"} default button 1 with icon note
		return
	end if

	set repo to "com.github.abakum.drt"
	tell application "Terminal"
		--When the script starts, Terminal sometimes creates an empty first window.
		--The idea is to open the script in the first window if it doesn't already contain this script,
		--and if it does contain this script, to open a new window instead.
		
		if (exists window 1) and not (custom title of first tab of window 1 is repo) then
			try
				do script shellCmd in window 1
			on error
				do script shellCmd
			end try
		else
			do script shellCmd
		end if

		set custom title of first tab of front window to repo
		activate
	end tell
end process

on joinList(theList, delimiter)
	set AppleScript's text item delimiters to delimiter
	set resultString to theList as text
	set AppleScript's text item delimiters to ""
	return resultString
end joinList