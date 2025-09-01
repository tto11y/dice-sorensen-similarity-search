package models

type MarkdownMeta struct {
	Model
	Name      string `gorm:"not null;unique" json:"name"`
	Path      string `gorm:"not null" json:"path"`
	CharCount uint   `gorm:"not null;default:0" json:"-"`
}

type MarkdownContent struct {
	Model
	Content string       `gorm:"not null" json:"content"`
	MetaID  uint         `json:"metaId" gorm:"not null;unique;foreignKey:MetaID;references:ID"`
	Meta    MarkdownMeta `json:"markdownFile"`
}
