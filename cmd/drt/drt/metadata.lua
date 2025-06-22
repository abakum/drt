-- Глобальная переменная resolve, как вы просили
resolve = Resolve()

-- Остальные переменные локальные
local projectManager = resolve:GetProjectManager()
local project = projectManager:GetCurrentProject()
local mediaPool = project:GetMediaPool()
local rootFolder = mediaPool:GetRootFolder()
local clips = rootFolder:GetClipList()

-- Функция для получения корневой папки и списка таймлайнов
local function rootTimeLines()
    local root = ""
    local timeLines = {}
    
    for _, clip in ipairs(clips) do
        local filePath = clip:GetClipProperty("File Path")
        if filePath == "" then
            -- Это таймлайн
            table.insert(timeLines, clip)
        else
            -- В моих проектах все клипы в одном каталоге
            -- Получаем директорию из пути к файлу
            root = string.match(filePath, "(.*[/\\])") or ""
            root = string.sub(root, 1, -2) -- Удаляем последний слеш
        end
    end
    
    return root, timeLines
end

-- Функция для очистки имени файла
local function sanitizeFilename(filename)
    -- Удаляем недопустимые символы в именах файлов
    local invalidChars = '[<>:"/\\|?*]'
    local sanitized = string.gsub(filename, invalidChars, "_")
    return sanitized
end

-- Основная функция
local function main()
    -- Получаем корневую папку и таймлайны
    local root, timeLines = rootTimeLines()
    print("Root path:", root)
    print("Number of timelines:", #timeLines)
    
    -- Таблица для хранения клипов по таймлайнам и mediaId
    local tlm = {} -- структура: tlm[timelineName][mediaId] = mediaPoolItem
    
    -- Получаем количество таймлайнов в проекте
    local tlc = project:GetTimelineCount()
    
    -- Перебираем все таймлайны
    for i = 1, tlc do
        local tl = project:GetTimelineByIndex(i)
        local tln = tl:GetName()
        
        -- Инициализируем таблицу для этого таймлайна
        if not tlm[tln] then
            tlm[tln] = {}
        end
        
        -- Проверяем все типы треков
        local trackTypes = {"subtitle", "video", "audio"}
        for _, trackType in ipairs(trackTypes) do
            local tc = tl:GetTrackCount(trackType)
            
            -- Перебираем все треки этого типа
            for j = 1, tc do
                local tlis = tl:GetItemListInTrack(trackType, j)
                
                -- Перебираем все элементы в треке
                for _, tli in ipairs(tlis) do
                    local mpi = tli:GetMediaPoolItem()
                    local mi = mpi:GetMediaId()
                    
                    -- Сохраняем клип по mediaId
                    tlm[tln][mi] = mpi
                    
                    print(string.format("Timeline: %s | Clip: %s", 
                          tln, mpi:GetName()))
                end
            end
        end
    end
    
    -- Экспортируем метаданные для каждого таймлайна
    for _, timeLine in ipairs(timeLines) do
        local name = sanitizeFilename(timeLine:GetName())
        local file = root .. "/" .. name .. ".csv"
        
        -- Собираем клипы для экспорта (только из текущего таймлайна)
        local exportClips = {timeLine}
        local tln = timeLine:GetName()
        
        if tlm[tln] then
            for mi, mpi in pairs(tlm[tln]) do
                table.insert(exportClips, mpi)
            end
        end
        
        -- Экспортируем метаданные
        print("Exporting metadata to:", file)
        mediaPool:ExportMetadata(file, exportClips)
    end
end

-- Запускаем основную функцию
main()