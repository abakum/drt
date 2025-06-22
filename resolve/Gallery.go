package resolve

type Gallery struct{}

// 1. GetAlbumName - возвращает имя альбома стиллов
func (g *Gallery) GetAlbumName(album *GalleryStillAlbum) string {
	return ""
}

// 2. SetAlbumName - устанавливает имя альбома стиллов
func (g *Gallery) SetAlbumName(album *GalleryStillAlbum, albumName string) bool {
	return false
}

// 3. GetCurrentStillAlbum - возвращает текущий альбом стиллов
func (g *Gallery) GetCurrentStillAlbum() *GalleryStillAlbum {
	return nil
}

// 4. SetCurrentStillAlbum - устанавливает текущий альбом стиллов
func (g *Gallery) SetCurrentStillAlbum(album *GalleryStillAlbum) bool {
	return false
}

// 5. GetGalleryStillAlbums - возвращает список всех альбомов стиллов
func (g *Gallery) GetGalleryStillAlbums() []*GalleryStillAlbum {
	return nil
}
