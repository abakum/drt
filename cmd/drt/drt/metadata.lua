-- Глобальная переменная resolve, как вы просили
resolve = Resolve()
-- Остальные переменные локальные
local projectManager = resolve:GetProjectManager()
local project = projectManager:GetCurrentProject()
local mediaPool = project:GetMediaPool()
local rootFolder = mediaPool:GetRootFolder()
local clips = rootFolder:GetClipList()
local exported = 0

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
    print("=== Экспорт метаданных ===")
    assert(project, "Проект не найден")

    -- Получаем корневую папку и таймлайны
    local outputFolder, timeLines = rootTimeLines()
    assert(outputFolder ~= "","Пустой медиапул")
    assert(#timeLines ~= 0 ,"Нет таймлайнов")

    
    -- Таблица для хранения клипов по таймлайнам и mediaId
    local tlm = {} -- структура: tlm[timelineName][mediaId] = mediaPoolItem
    
    -- Получаем количество таймлайнов в проекте
    local tlc = project:GetTimelineCount()
    
    -- Перебираем все таймлайны
    for i = 1, tlc do
        local tl = project:GetTimelineByIndex(i)
        local tln = tl:GetName()

        -- Экспортируем в .drt
        -- local name = sanitizeFilename(tln)
        -- local file = outputFolder .. "/" .. name .. ".drt"
        -- tl:Export(file, resolve.EXPORT_DRT, resolve.EXPORT_NONE)

        
        -- Инициализируем таблицу для этого таймлайна
        if not tlm[tln] then
            tlm[tln] = {}
        end
        
        -- по типам треков
        local trackTypes = {"video", "audio"}
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
                end
            end
        end
    end
    
    -- Экспортируем метаданные для каждого таймлайна
    for _, timeLine in ipairs(timeLines) do
        local tln = timeLine:GetName()
        local name = sanitizeFilename(tln)
        local file = outputFolder .. "/" .. name .. ".csv"
        
        -- Собираем клипы для экспорта (только из текущего таймлайна)
        local exportClips = {timeLine}
        
        if tlm[tln] then
            for mi, mpi in pairs(tlm[tln]) do
                table.insert(exportClips, mpi)
            end
        end
        
        -- Экспортируем метаданные
        print("Exporting metadata to:", file)
        if mediaPool:ExportMetadata(file, exportClips) then
            exported = exported+1
        end
    end
    print(string.format("=== Экспортировано %d из %d ===", exported,  tlc))
end

-- Запускаем основную функцию
main()