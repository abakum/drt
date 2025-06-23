resolve = Resolve()
local projectManager = resolve:GetProjectManager()
local project = projectManager:GetCurrentProject()

if not project then
    print("Нет открытого проекта")
    return
end

local timeline = project:GetCurrentTimeline()

if not timeline then
    print("Таймлайн не найден")
    return
end

-- Функция преобразования номера кадра в временной код
local function frameToTimecode(frame, framerate)
    local fps = framerate or timeline:GetSetting("timelineFrameRate")
    local hours = math.floor(frame / (fps * 3600))
    local remaining = frame % (fps * 3600)
    local minutes = math.floor(remaining / (fps * 60))
    remaining = remaining % (fps * 60)
    local seconds = math.floor(remaining / fps)
    local frames = remaining % fps
    
    return string.format("%02d:%02d:%02d:%02d", hours, minutes, seconds, frames)
end

-- Получаем маркеры
local markers = timeline:GetMarkers()

if not markers or not next(markers) then
    print("Маркеры не найдены")
    return
end

-- Создаем содержимое EDL
local edlContent = "TITLE: Resolve Markers Export\n"
edlContent = edlContent .. "FCM: NON-DROP FRAME\n\n"

local framerate = timeline:GetSetting("timelineFrameRate")
local counter = 1

for frame, marker in pairs(markers) do
    local timecode = frameToTimecode(frame, framerate)
    local duration = marker.duration or 0
    local color = marker.color or "Green"
    local comment = marker.note or ""
    
    edlContent = edlContent .. string.format(
        "%03d  AX       V     C        %s %s %s %s\n",
        counter,
        timecode,
        frameToTimecode(frame + duration, framerate),
        timecode,
        frameToTimecode(frame + duration, framerate)
    )
    
    edlContent = edlContent .. string.format(
        "* FROM CLIP NAME: %s\n* %s\n* COLOR: %s\n\n",
        comment,
        comment,
        color
    )
    
    counter = counter + 1
end

-- Сохраняем в файл
local filePath = "C:/markers_export.edl"  -- Измените на нужный путь
local file = io.open(filePath, "w")

if file then
    file:write(edlContent)
    file:close()
    print("Маркеры успешно экспортированы в " .. filePath)
else
    print("Ошибка создания файла")
end