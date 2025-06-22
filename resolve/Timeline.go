package resolve

type Timeline struct{}

// 1. GetName - возвращает имя таймлайна
func (t *Timeline) GetName() string {
	return ""
}

// 2. SetName - устанавливает имя таймлайна
func (t *Timeline) SetName(name string) bool {
	return false
}

// 3. GetStartFrame - возвращает начальный кадр
func (t *Timeline) GetStartFrame() int {
	return 0
}

// 4. GetEndFrame - возвращает конечный кадр
func (t *Timeline) GetEndFrame() int {
	return 0
}

// 5. SetStartTimecode - устанавливает стартовый таймкод
func (t *Timeline) SetStartTimecode(timecode string) bool {
	return false
}

// 6. GetStartTimecode - возвращает стартовый таймкод
func (t *Timeline) GetStartTimecode() string {
	return ""
}

// 7. GetTrackCount - возвращает количество треков
func (t *Timeline) GetTrackCount(trackType string) int {
	return 0
}

// 8. GetItemListInTrack - возвращает элементы трека
func (t *Timeline) GetItemListInTrack(trackType string, index int) []*TimelineItem {
	return nil
}

// 9. AddMarker - добавляет маркер
func (t *Timeline) AddMarker(frameId int, color, name, note string, duration int, customData string) bool {
	return false
}

// 10. GetMarkers - возвращает все маркеры
func (t *Timeline) GetMarkers() map[int]Marker {
	return nil
}

// 11. GetMarkerByCustomData - возвращает маркер по данным
func (t *Timeline) GetMarkerByCustomData(customData string) map[int]Marker {
	return nil
}

// 12. UpdateMarkerCustomData - обновляет данные маркера
func (t *Timeline) UpdateMarkerCustomData(frameId int, customData string) bool {
	return false
}

// 13. GetMarkerCustomData - возвращает данные маркера
func (t *Timeline) GetMarkerCustomData(frameId int) string {
	return ""
}

// 14. DeleteMarkersByColor - удаляет маркеры по цвету
func (t *Timeline) DeleteMarkersByColor(color string) bool {
	return false
}

// 15. DeleteMarkerAtFrame - удаляет маркер на кадре
func (t *Timeline) DeleteMarkerAtFrame(frameNum int) bool {
	return false
}

// 16. DeleteMarkerByCustomData - удаляет маркер по данным
func (t *Timeline) DeleteMarkerByCustomData(customData string) bool {
	return false
}

// 17. ApplyGradeFromDRX - применяет градацию из файла
func (t *Timeline) ApplyGradeFromDRX(path string, gradeMode int, items ...*TimelineItem) bool {
	return false
}

// 18. GetCurrentTimecode - возвращает текущий таймкод
func (t *Timeline) GetCurrentTimecode() string {
	return ""
}

// 19. SetCurrentTimecode - устанавливает текущий таймкод.
// timecode format - defaults to "00:00:00:00"
func (t *Timeline) SetCurrentTimecode(timecode string) bool {
	return false
}

// 20. GetCurrentVideoItem - возвращает текущий видео элемент
func (t *Timeline) GetCurrentVideoItem() *TimelineItem {
	return nil
}

// 21. GetCurrentClipThumbnailImage - возвращает thumbnail
func (t *Timeline) GetCurrentClipThumbnailImage() map[string]interface{} {
	return nil
}

// 22. GetTrackName - возвращает имя трека
func (t *Timeline) GetTrackName(trackType string, trackIndex int) string {
	return ""
}

// 23. SetTrackName - устанавливает имя трека
func (t *Timeline) SetTrackName(trackType string, trackIndex int, name string) bool {
	return false
}

// 24. DuplicateTimeline - дублирует таймлайн
func (t *Timeline) DuplicateTimeline(name string) *Timeline {
	return nil
}

// 25. CreateCompoundClip - создает составной клип
func (t *Timeline) CreateCompoundClip(items []*TimelineItem, clipInfo map[string]string) *TimelineItem {
	return nil
}

// 26. CreateFusionClip - создает Fusion клип
func (t *Timeline) CreateFusionClip(items []*TimelineItem) *TimelineItem {
	return nil
}

// 27. ImportIntoTimeline - импортирует в таймлайн
func (t *Timeline) ImportIntoTimeline(filePath string, importOptions map[string]interface{}) bool {
	return false
}

// 28. Export - экспортирует таймлайн
func (t *Timeline) Export(fileName string, exportType, exportSubtype int) bool {
	return false
}

// 29. GetSetting - возвращает настройку
func (t *Timeline) GetSetting(settingName string) string {
	return ""
}

// 30. SetSetting - устанавливает настройку
func (t *Timeline) SetSetting(settingName, settingValue string) bool {
	return false
}

// 31. InsertGeneratorIntoTimeline - вставляет генератор
func (t *Timeline) InsertGeneratorIntoTimeline(generatorName string) *TimelineItem {
	return nil
}

// 32. InsertFusionGeneratorIntoTimeline - вставляет Fusion генератор
func (t *Timeline) InsertFusionGeneratorIntoTimeline(generatorName string) *TimelineItem {
	return nil
}

// 33. InsertFusionCompositionIntoTimeline - вставляет Fusion композицию
func (t *Timeline) InsertFusionCompositionIntoTimeline() *TimelineItem {
	return nil
}

// 34. InsertOFXGeneratorIntoTimeline - вставляет OFX генератор
func (t *Timeline) InsertOFXGeneratorIntoTimeline(generatorName string) *TimelineItem {
	return nil
}

// 35. InsertTitleIntoTimeline - вставляет заголовок
func (t *Timeline) InsertTitleIntoTimeline(titleName string) *TimelineItem {
	return nil
}

// 36. InsertFusionTitleIntoTimeline - вставляет Fusion заголовок
func (t *Timeline) InsertFusionTitleIntoTimeline(titleName string) *TimelineItem {
	return nil
}

// 37. GrabStill - захватывает still
func (t *Timeline) GrabStill() *GalleryStill {
	return nil
}

// 38. GrabAllStills - захватывает все stills
func (t *Timeline) GrabAllStills(stillFrameSource int) []*GalleryStill {
	return nil
}
