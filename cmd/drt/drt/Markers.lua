-- Объявление глобальных переменных
resolve = Resolve()
ProjectManager = resolve:GetProjectManager()
Project = ProjectManager:GetCurrentProject()
MediaPool = Project:GetMediaPool()
RootFolder = MediaPool:GetRootFolder()
Clips = RootFolder:GetClipList()
OutputFolder = ""
-- FrameRate = 0.0
Exported = 0
Total = 0
RenderJobsAdded = false

-- Константы
local FORMAT = "png"
local CODEC = "RGB8"
local FORMATb = "tif"
local CODECb = "RGB8LZW"

--https://github.com/rogmag/rogmag.github.io/blob/main/downloads/creating_davinci_resolve_scripts/scripts/utility/libavutil/Libavutil.lua
local luaresolve, libavutil
luaresolve = 
{
	frame_rates =
	{
		get_fraction = function(self, frame_rate_string_or_number)
			local frame_rate = tonumber(tostring(frame_rate_string_or_number))
			-- These are the frame rates that DaVinci Resolve Studio supports as of version 18
			local frame_rates = { 16, 18, 23.976, 24, 25, 29.97, 30, 47.952, 48, 50, 59.94, 60, 72, 95.904, 96, 100, 119.88, 120 }

			for _, current_frame_rate in ipairs (frame_rates) do
				if current_frame_rate == frame_rate or math.floor(current_frame_rate) == frame_rate then
					local is_decimal = current_frame_rate % 1 > 0
					local denominator = iif(is_decimal, 1001, 100)
					local numerator = math.ceil(current_frame_rate) * iif(is_decimal, 1000, denominator)
					return { num = numerator, den = denominator }
				end
			end

			return nil, string.format("Invalid frame rate: %s", frame_rate_string_or_number)
		end,

		get_decimal = function(self, frame_rate_string_or_number)
			local fractional_frame_rate, error_message = self:get_fraction(frame_rate_string_or_number)
			
			if fractional_frame_rate ~= nil then
				return tonumber(string.format("%.3f", fractional_frame_rate.num / fractional_frame_rate.den))
			else
				return nil, error_message
			end
		end,
	},

	load_library = function(name_pattern)
		local files = bmd.readdir(fu:MapPath("FusionLibs:"..iif(ffi.os == "Windows", "", "../"))..name_pattern)
		assert(#files == 1 and files[1].IsDir == false, string.format("Couldn't find exact match for pattern \"%s.\"", name_pattern))
		return ffi.load(files.Parent..files[1].Name)
	end,

	frame_from_timecode = function(self, timecode, frame_rate)
		return libavutil:av_timecode_init_from_string(timecode, self.frame_rates:get_fraction(frame_rate)).start
	end,

	timecode_from_frame = function(self, frame, frame_rate, drop_frame)
		return libavutil:av_timecode_make_string(0, frame, self.frame_rates:get_decimal(frame_rate),
		{
			AV_TIMECODE_FLAG_DROPFRAME = drop_frame == true or drop_frame == 1 or drop_frame == "1",
			AV_TIMECODE_FLAG_24HOURSMAX = true,
			AV_TIMECODE_FLAG_ALLOWNEGATIVE = false
		})
	end
}

libavutil = 
{
	library = luaresolve.load_library(iif(ffi.os == "Windows", "avutil*.dll", iif(ffi.os == "OSX", "libavutil*.dylib", "libavutil.so"))),

	demand_version = function(self, version)
		local library_version = self:av_version_info()

		return (library_version.major > version.major)
			or (library_version.major == version.major and library_version.minor > version.minor)
			or (library_version.major == version.major and library_version.minor == version.minor and library_version.patch > version.patch)
			or (library_version.major == version.major and library_version.minor == version.minor and library_version.patch == version.patch)
	end,

	set_declarations = function()
		ffi.cdef[[
			enum AVTimecodeFlag {
				AV_TIMECODE_FLAG_DROPFRAME      = 1<<0, // timecode is drop frame
				AV_TIMECODE_FLAG_24HOURSMAX     = 1<<1, // timecode wraps after 24 hours
				AV_TIMECODE_FLAG_ALLOWNEGATIVE  = 1<<2, // negative time values are allowed
			};

			struct AVRational { int32_t num; int32_t den; };
			struct AVTimecode { int32_t start; enum AVTimecodeFlag flags; struct AVRational rate; uint32_t fps; };

			char* av_timecode_make_string(const struct AVTimecode* tc, const char* buf, int32_t framenum);
			int32_t av_timecode_init_from_string(struct AVTimecode* tc, struct AVRational rate, const char* str, void* log_ctx);

			char* av_version_info (void);
		]]
	end,

	av_timecode_make_string = function(self, start, frame, fps, flags)
		local function bor_number_flags(enum_name, flags)
			local enum_value = 0    
	
			if (flags) then
				for key, value in pairs(flags) do
					if (value == true) then
						enum_value = bit.bor(enum_value, tonumber(ffi.new(enum_name, key)))
					end
				end
			end

			return enum_value;
		end

		local tc = ffi.new("struct AVTimecode",
		{
			start = start,
			flags = bor_number_flags("enum AVTimecodeFlag", flags),
			fps = math.ceil(luaresolve.frame_rates:get_decimal(fps))
		})

		if (flags.AV_TIMECODE_FLAG_DROPFRAME and fps > 60 and (fps % (30000 / 1001) == 0 or fps % 29.97 == 0))
			and (not self:demand_version( { major = 4, minor = 4, patch = 0 } ))
		then
			-- Adjust for drop frame above 60 fps (not necessary if BMD upgrades to libavutil-57 or later)
			frame = frame + 9 * tc.fps / 15 * (math.floor(frame / (tc.fps * 599.4))) + (math.floor((frame % (tc.fps * 599.4)) / (tc.fps * 59.94))) * tc.fps / 15
		end

		local timecodestring = ffi.string(self.library.av_timecode_make_string(tc, ffi.string(string.rep(" ", 16)), frame))
	
		if (#timecodestring > 0) then
			local frame_digits = #tostring(math.ceil(fps) - 1)

			-- Fix for libavutil where it doesn't use leading zeros for timecode at frame rates above 100
			if frame_digits > 2 then
				timecodestring = string.format("%s%0"..frame_digits.."d", timecodestring:sub(1, timecodestring:find("[:;]%d+$")), tonumber(timecodestring:match("%d+$")))
			end

			return timecodestring
		else
			return nil
		end
	end,

	av_timecode_init_from_string = function(self, timecode, frame_rate_fraction)
		local tc = ffi.new("struct AVTimecode")
		local result = self.library.av_timecode_init_from_string(tc, ffi.new("struct AVRational", frame_rate_fraction), timecode, ffi.new("void*", nil))
	
		if (result == 0) then
			return
			{
				start = tc.start,
				flags =
				{
					AV_TIMECODE_FLAG_DROPFRAME = bit.band(tc.flags, ffi.C.AV_TIMECODE_FLAG_DROPFRAME) == ffi.C.AV_TIMECODE_FLAG_DROPFRAME,
					AV_TIMECODE_FLAG_24HOURSMAX = bit.band(tc.flags, ffi.C.AV_TIMECODE_FLAG_24HOURSMAX) == ffi.C.AV_TIMECODE_FLAG_24HOURSMAX,
					AV_TIMECODE_FLAG_ALLOWNEGATIVE = bit.band(tc.flags, ffi.C.AV_TIMECODE_FLAG_ALLOWNEGATIVE) == ffi.C.AV_TIMECODE_FLAG_ALLOWNEGATIVE,
				},
				rate =
				{
					num = tc.rate.num,
					den = tc.rate.den
				},
				fps = tc.fps
			}
		else
			error("avutil error code: "..result)
		end
	end,

	av_version_info = function(self)
		local version = ffi.string(self.library.av_version_info())

		return 
		{
			major = tonumber(version:match("^%d+")),
			minor = tonumber(version:match("%.%d+"):sub(2)),
			patch = tonumber(version:match("%d+$"))
		}
	end,
}

libavutil.set_declarations()

-- Функция преобразования времени в таймкод
-- local function timeToTimecode(seconds, fps)
--     return string.format("%02d:%02d:%02d:%02d",
--         math.floor(seconds / 3600),
--         math.floor(math.fmod(seconds, 3600) / 60),
--         math.floor(math.fmod(seconds, 60)),
--         math.floor(math.fmod(seconds, 1) * fps))
-- end


-- Функция очистки имени файла
local function sanitizeFilename(name)
    if name == "" then
        return "frame"
    end
    
    -- Замена недопустимых символов
    name = string.gsub(name, '[\\/:*?"<>|]', '_')
    -- Удаление лишних пробелов
    name = string.gsub(name, '%s+', ' ')
    return string.gsub(name, '^%s*(.-)%s*$', '%1')
end

-- Функция экспорта кадра
local function exportFrameAsStill(pos, outputPath)
    -- Попробуем новый метод
    if type(Project.ExportCurrentFrameAsStill) == "function" then
        if Project:ExportCurrentFrameAsStill(outputPath) then
            return true
        end
        if resolve:GetCurrentPage() == "deliver" then
            print("С панели Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
        end
    end

    -- План B через Deliver
    local timeline = Project:GetCurrentTimeline()
    local startFrame = timeline:GetStartFrame()

    local renderSettings = {
        MarkIn = startFrame + pos,
        MarkOut = startFrame + pos,
        CustomName = string.gsub(outputPath:match("([^/\\]+)$"), "%..+$", ""),
        TargetDir = OutputFolder,
        ExportVideo = false,
        ExportAudio = false
    }

    if not Project:SetRenderSettings(renderSettings) then
        print("Не удалось установить настройки рендера")
        return false
    end

    if not Project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
        if not Project:SetCurrentRenderFormatAndCodec(FORMATb, CODECb) then
            print(string.format("Не удалось установить формат %s и кодек %s", FORMATb, CODECb))
            return false
        end
    end

    local jobId = Project:AddRenderJob()
    if jobId ~= "" then
        RenderJobsAdded =true
    end
    return true
end

-- Функция экспорта помеченных кадров
local function exportMarkedFrames()
    local timeline = Project:GetCurrentTimeline()
    local timelineName = sanitizeFilename(timeline:GetName())
    local markers = timeline:GetMarkers()
    
    if not markers or type(markers) ~= "table" or not next(markers) then
        print("Не найдены маркеры в " .. timelineName)
        return
    end


    -- Сортировка маркеров
    local positions = {}
    for pos, _ in pairs(markers) do
        table.insert(positions, pos)
    end
    table.sort(positions)

    -- Экспорт кадров
    for _, pos in ipairs(positions) do
        local marker = markers[pos]
        local name = marker.name
        if name == "Marker 1" then
            name = ""
        end

        --local timecode = timeToTimecode(pos / FrameRate, FrameRate)
        
        -- https://github.com/rogmag/rogmag.github.io/blob/main/downloads/creating_davinci_resolve_scripts/scripts/utility/libavutil/Libavutil.lua
        local startFrame = timeline:GetStartFrame()
        local frame_rate = luaresolve.frame_rates:get_decimal(timeline:GetSetting("timelineFrameRate"))
        local drop_frame = timeline:GetSetting("timelineDropFrameTimecode") == "1"
        local frame = startFrame + pos
        local timecode = luaresolve:timecode_from_frame(frame, frame_rate, drop_frame)
        timeline:SetCurrentTimecode(timecode)

        local filename = timelineName .. name
        local fullPath = OutputFolder .. "/" .. filename .. "." .. FORMAT

        if exportFrameAsStill(pos, fullPath) then
            print(string.format("Экспортирован %s -> %s", timecode, filename))
            Exported = Exported + 1
        else
            print("Ошибка экспорта " .. timecode)
        end
        Total = Total + 1
    end
end

-- Функция экспорта всех таймлайнов
local function exportAllTimelines()
    local timelineCount = Project:GetTimelineCount()

    -- Сохраняем оригинальный таймлайн
    local originalTimeline = Project:GetCurrentTimeline()

    -- Обрабатываем все таймлайны
    for i = 1, timelineCount do
        local timeline = Project:GetTimelineByIndex(i)
        Project:SetCurrentTimeline(timeline)
        exportMarkedFrames()
    end

    -- Восстанавливаем оригинальный таймлайн
    if timelineCount > 1 then
        Project:SetCurrentTimeline(originalTimeline)
    end
end

-- Функция получения корневой папки и подсчета маркеров
local function root()
    
    for _, clip in ipairs(Clips) do
        local filePath = clip:GetClipProperty("File Path")
        if filePath ~= "" then
            -- Допустим исходные и результирующие медиафайлы в одном каталоге
            return string.match(filePath, "(.*[/\\])") or ""
        end
    end
    
    return ""
end

-- Основная функция
local function main()
    print("=== Экспорт кадров с маркерами ===")

    if not Project then
        print("Проект не найден")
        return
    end
    
    OutputFolder = root()
    if OutputFolder == "" then
        print("Пустой медиапул")
        return
    end
    
    -- local fpsStr = Project:GetSetting("timelineFrameRate")
    -- FrameRate = tonumber(fpsStr) or 24.0

    Project:DeleteAllRenderJobs()
    if resolve:GetCurrentPage() == "edit" then
		print("С панели Edit экспорт кадров с маркерами для одного таймлайна")
        exportMarkedFrames()
    else
        exportAllTimelines()
    end


    print(string.format("=== Экспортировано %d из %d ===", Exported, Total))
    if RenderJobsAdded then
        Project:StartRendering()
    end
end

-- Запуск программы
main()