on run {input, parameters}    

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

    set bundleID to "com.github.abakum.drt"
    tell application "Terminal"
        --When the script starts, Terminal sometimes creates an empty first window.
        --The idea is to open the script in the first window if it doesn't already contain this script,
        --and if it does contain this script, to open a new window instead.

        if (exists window 1) and not (custom title of first tab of window 1 is bundleID) then
            try
                do script shellCmd in window 1
            on error
                do script shellCmd
            end try
        else
            do script shellCmd
        end if

        set custom title of first tab of front window to bundleID
        activate
    end tell

    return input
end run