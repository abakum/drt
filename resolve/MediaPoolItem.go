// https://resolvedevdoc.readthedocs.io/en/latest/API_basic.html#mediapoolitem
package resolve

type MediaPoolItem struct{} // Элемент медиапула

// 1-3. Основные свойства
func (i *MediaPoolItem) GetName() string                     { return "" }
func (i *MediaPoolItem) GetMediaId() string                  { return "" }
func (i *MediaPoolItem) GetClipProperty(prop string) string  { return "" }
func (i *MediaPoolItem) GetClipPropertys() map[string]string { return nil }

// 4-6. Метаданные
func (i *MediaPoolItem) GetMetadata(key string) interface{}          { return nil }
func (i *MediaPoolItem) SetMetadata(key, value string) bool          { return false }
func (i *MediaPoolItem) SetMetadataBulk(data map[string]string) bool { return false }

// 7-16. Маркеры (10 методов)
func (i *MediaPoolItem) AddMarker(frame int, color, name, note string, duration int, data string) bool {
	return false
}
func (i *MediaPoolItem) GetMarkers() map[int]Marker                         { return nil }
func (i *MediaPoolItem) GetMarkerByCustomData(data string) map[int]Marker   { return nil }
func (i *MediaPoolItem) UpdateMarkerCustomData(frame int, data string) bool { return false }
func (i *MediaPoolItem) GetMarkerCustomData(frame int) string               { return "" }
func (i *MediaPoolItem) DeleteMarkersByColor(color string) bool             { return false }
func (i *MediaPoolItem) DeleteMarkerAtFrame(frame int) bool                 { return false }
func (i *MediaPoolItem) DeleteMarkerByCustomData(data string) bool          { return false }
func (i *MediaPoolItem) GetStartFrame() int                                 { return 0 } // Доп. методы для таймлайна
func (i *MediaPoolItem) GetEndFrame() int                                   { return 0 }

// 17-19. Флаги (3 метода)
func (i *MediaPoolItem) AddFlag(color string) bool    { return false }
func (i *MediaPoolItem) GetFlagList() []string        { return nil }
func (i *MediaPoolItem) ClearFlags(color string) bool { return false }

// 20-22. Цвета клипов (3 метода)
func (i *MediaPoolItem) GetClipColor() string           { return "" }
func (i *MediaPoolItem) SetClipColor(color string) bool { return false }
func (i *MediaPoolItem) ClearClipColor() bool           { return false }

// 23-24. Прокси и замена (2 метода)
func (i *MediaPoolItem) LinkProxyMedia(path string) bool { return false }
func (i *MediaPoolItem) UnlinkProxyMedia() bool          { return false }

// 25-26. Дополнительные (из вашего примера)
func (i *MediaPoolItem) ReplaceClip(path string) bool                        { return false }
func (i *MediaPoolItem) SetClipProperty(prop string, value interface{}) bool { return false }

// Вспомогательные структуры
type Marker struct {
	Frame      int
	Color      string
	Name       string
	Note       string
	Duration   int
	CustomData string
}
