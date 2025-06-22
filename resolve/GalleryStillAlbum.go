package resolve

// Вспомогательные типы
type GalleryStillAlbum struct{}

// 1. GetStills - возвращает список стиллов в альбоме
func (gsa *GalleryStillAlbum) GetStills() []*GalleryStill {
	return nil
}

// 2. GetLabel - возвращает метку стилла
func (gsa *GalleryStillAlbum) GetLabel(still *GalleryStill) string {
	return ""
}

// 3. SetLabel - устанавливает метку стилла
func (gsa *GalleryStillAlbum) SetLabel(still *GalleryStill, label string) bool {
	return false
}

// 4. ExportStills - экспортирует стиллы в файлы
func (gsa *GalleryStillAlbum) ExportStills(stills []*GalleryStill, folderPath, filePrefix, format string) bool {
	return false
}

// 5. DeleteStills - удаляет указанные стиллы
func (gsa *GalleryStillAlbum) DeleteStills(stills []*GalleryStill) bool {
	return false
}

// Вспомогательные типы
type GalleryStill struct{}
