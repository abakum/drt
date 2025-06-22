resolve = Resolve()
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

-- Основная функция
local function main()
    -- Получаем корневую папку и таймлайны
    local root, timeLines = rootTimeLines()
    print("Root path:", root)
    print("Number of timelines:", #timeLines)
    
    -- Таблица для хранения клипов по таймлайнам
    local tlm = {} -- ключ: имя таймлайна, значение: список клипов
    
    -- Получаем количество таймлайнов в проекте
    local tlc = project:GetTimelineCount()
    
    -- Перебираем все таймлайны
    for i = 1, tlc do
        local tl = project:GetTimelineByIndex(i)
        
        -- Проверяем все типы треков
        local trackTypes = {"subtitle", "video", "audio"}
        for _, trackType in ipairs(trackTypes) do
            local tc = tl:GetTrackCount(trackType)
            
            -- Перебираем все треки этого типа
            for j = 1, tc do
                local tlis = tl:GetItemListInTrack(trackType, j)
                
                -- Перебираем все элементы в треке
                for _, tli in ipairs(tlis) do
                    local tln = tl:GetName()
                    print(string.format("Timeline %d (%s) - Track %s %d: %s", 
                          i, tln, trackType, j, tli:GetName()))
                    
                    -- Добавляем клип в таблицу
                    if not tlm[tln] then
                        tlm[tln] = {}
                    end
                    table.insert(tlm[tln], tli:GetMediaPoolItem())
                end
            end
        end
    end
    
    -- Экспортируем метаданные для каждого таймлайна
    for _, timeLine in ipairs(timeLines) do
        local name = timeLine:GetName()
        local file = root .. "\\" .. name .. ".csv"  -- Формируем путь к файлу CSV
        local tln = timeLine:GetName()
        
        print("Exporting metadata to:", file)
        print("Clips in timeline:", #(tlm[tln] or {}))
        
        -- Собираем все клипы для экспорта
        local exportItems = {}
        if tlm[tln] then
            for _, item in ipairs(tlm[tln]) do
                table.insert(exportItems, item)
            end
        end
        table.insert(exportItems, timeLine)
        
        -- Экспортируем метаданные
        mediaPool:ExportMetadata(file, exportItems)
    end
end

-- Запускаем основную функцию
main()