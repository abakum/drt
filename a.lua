--[[
 * Resolve Script Name: Export Poster Markers as Still Frames
 * Author: abakum
 * Licence: GPL v3
 * Version: 1.0
--]]

print("Начало экспорта кадров с маркерами")

function AddLeadingZeros(int)
    return string.format("%02d", tostring(int))
end

function GetTimeCodeFromFrame(pos, fps)
    local seconds = pos/fps
    local h = AddLeadingZeros(math.floor(seconds/3600))
    local m = AddLeadingZeros(math.floor((seconds % 3600)/60))
    local s = AddLeadingZeros(math.floor(seconds % 60))
    return h .. ":" .. m .. ":" .. s
end

-- Основная функция экспорта
function ExportPosterFrames()
    -- Инициализация Resolve
    resolve = Resolve()

    local ms = resolve:GetMediaStorage()
    --local cf = ms:GetCurrentDirectory()

    local pm = resolve:GetProjectManager()
    local proj = pm:GetCurrentProject()
    
    if not proj then
        print("Ошибка: Нет открытого проекта")
        return
    end

    local tl = proj:GetCurrentTimeline()
    if not tl then
        print("Ошибка: Нет открытого таймлайна")
        return
    end

    -- Получаем настройки проекта
    local framerate = proj:GetSetting("timelineFrameRate")
    local markers = tl:GetMarkers()

    -- Создаем папку для экспорта
    local output_folder = 'c:/tmp/'
   -- local os_cmd = 'mkdir "' .. output_folder .. '" 2>nul'
   -- os.execute(os_cmd)

    -- Сортируем маркеры по позиции
    local positions = {}
    for pos, marker in pairs(markers) do
        table.insert(positions, pos)
    end
    table.sort(positions)

    -- Обрабатываем маркеры
    local exported_count = 0
    for i, pos in ipairs(positions) do
        local marker = markers[pos]
        if marker.name == "Marker 1" then
            -- Устанавливаем текущий кадр
            tl:SetCurrentTimecode(pos)

            -- Пример использования
            local clipPath = GetCurrentClipPath()
          
            if clipPath then
                output_folder=GetParentDirectory(clipPath)
                print("Путь к текущему клипу: " .. clipPath)
            else
                print("Клип в текущей позиции не найден")
            end

            -- Генерируем имя файла
            local timecode = GetTimeCodeFromFrame(pos, framerate)
            local filename = tl:GetName() .. ".png"
            
            -- Экспортируем кадр
            local success = proj:ExportCurrentFrameAsStill(output_folder .. filename)
            
            if success then
                print("Экспортирован: " .. timecode .. " -> " .. filename)
                exported_count = exported_count + 1
            else
                print("Ошибка экспорта кадра: " .. timecode)
            end
        end
    end

    print("\nГотово! Экспортировано кадров: " .. exported_count)
end

function GetCurrentClipPath()
    resolve = Resolve()
    local project = resolve:GetProjectManager():GetCurrentProject()
    if not project then return nil end
    
    local timeline = project:GetCurrentTimeline()
    if not timeline then return nil end
    
    -- Получаем frame rate проекта
    local frameRate = tonumber(project:GetSetting("timelineFrameRate")) or 24
    
    -- Получаем текущую позицию (может вернуть число кадров ИЛИ строку таймкода)
    local currentPos = timeline:GetCurrentTimecode()
    local currentFrames
    
    if type(currentPos) == "number" then
        -- Уже получили кадры напрямую
        currentFrames = currentPos
    else
        -- Конвертируем из строки таймкода
        currentFrames = TimecodeToFrames(currentPos, frameRate)
    end
    
    -- Проверяем клипы на первом видео-треке
    local clips = timeline:GetItemListInTrack("video", 1)
    if not clips then return nil end
    
    for _, clip in ipairs(clips) do
        local clipStart = clip:GetStart()
        local clipDuration = clip:GetDuration()
        
        -- Получаем начальный кадр клипа
        local startFrames
        if type(clipStart) == "number" then
            startFrames = clipStart
        else
            startFrames = TimecodeToFrames(clipStart, frameRate)
        end
        
        -- Вычисляем конечный кадр клипа
        local endFrames = startFrames + clipDuration - 1
        
        if currentFrames >= startFrames and currentFrames <= endFrames then
            local mediaPoolItem = clip:GetMediaPoolItem()
            if mediaPoolItem then
                return mediaPoolItem:GetClipProperty("File Path")
            end
        end
    end
    
    return nil
end

-- Универсальная конвертация таймкода в кадры
function TimecodeToFrames(timecodeStr, frameRate)
    if type(timecodeStr) ~= "string" then return 0 end
    
    local h, m, s, f = timecodeStr:match("(%d+):(%d+):(%d+):(%d+)")
    h, m, s, f = tonumber(h), tonumber(m), tonumber(s), tonumber(f)
    
    return f + s * frameRate + m * 60 * frameRate + h * 3600 * frameRate
end

function GetParentDirectory(filepath)
    if not filepath or type(filepath) ~= "string" or filepath == "" then
        return nil
    end
    
    -- Нормализация разделителей
    filepath = filepath:gsub("\\", "/")
    
    -- Удаление завершающих слешей
    filepath = filepath:gsub("/+$", "")
    
    -- Извлечение родительской директории
    local parentDir = filepath:match("^(.*)/[^/]*$")
    
    -- Если не удалось извлечь (уже корневая директория)
    if not parentDir then
        return filepath
    end
    
    -- Восстановление оригинальных разделителей для Windows
    if filepath:match("^%a:/") then
        parentDir = parentDir:gsub("/", "\\")
        -- Добавляем обратный слеш если его нет
        if not parentDir:match("\\$") then
            parentDir = parentDir .. "\\"
        end
    end
    
    return parentDir
end

-- Запускаем основную функцию
ExportPosterFrames()