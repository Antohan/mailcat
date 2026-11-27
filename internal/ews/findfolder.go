package ews

// FindFolder — список папок от msgfolderroot (docs/EWS-REFERENCE.md §4).
// Read-only операция (AGENTS.md §1).

import (
	"context"
	"encoding/xml"
	"fmt"
)

// FolderID — идентификатор папки Exchange: Id + ChangeKey (ChangeKey нужен
// для последующих операций с той же версией папки).
type FolderID struct {
	ID        string
	ChangeKey string
}

// Folder — почтовая папка из ответа FindFolder.
type Folder struct {
	ID               string
	ChangeKey        string
	DisplayName      string
	TotalCount       int // всего писем
	UnreadCount      int // непрочитанных
	ChildFolderCount int // вложенных папок
}

// FindFolders возвращает папки верхнего уровня (Traversal="Shallow" от
// DistinguishedFolderId msgfolderroot). BaseShape Default уже содержит все
// поля Folder (EWS-REFERENCE.md §4).
func (c *Client) FindFolders(ctx context.Context) ([]Folder, error) {
	var resp findFolderResponse
	if err := c.Do(ctx, newFindFoldersRequest(), &resp); err != nil {
		return nil, fmt.Errorf("список папок: %w", err)
	}
	folders, err := resp.result()
	if err != nil {
		return nil, fmt.Errorf("список папок: %w", err)
	}
	return folders, nil
}

// newFindFoldersRequest строит запрос по шаблону EWS-REFERENCE.md §4.
func newFindFoldersRequest() *findFolderRequest {
	return &findFolderRequest{
		Traversal: "Shallow",
		FolderShape: folderShapeRequest{
			BaseShape: "Default",
		},
		ParentFolderIDs: parentFolderIDsRequest{
			DistinguishedFolderID: &distinguishedFolderIDRequest{ID: "msgfolderroot"},
		},
	}
}

// findFolderRequest — m:FindFolder (запрос; namespace-URL в тегах — литералы,
// выражения в struct-тегах Go не допускаются).
type findFolderRequest struct {
	XMLName         xml.Name               `xml:"http://schemas.microsoft.com/exchange/services/2006/messages FindFolder"`
	Traversal       string                 `xml:"Traversal,attr"`
	FolderShape     folderShapeRequest     `xml:"FolderShape"`
	ParentFolderIDs parentFolderIDsRequest `xml:"ParentFolderIds"`
}

// folderShapeRequest — m:FolderShape c t:BaseShape.
type folderShapeRequest struct {
	BaseShape string `xml:"http://schemas.microsoft.com/exchange/services/2006/types BaseShape"`
}

// parentFolderIDsRequest — m:ParentFolderIds: ровно одна ссылка на папку —
// либо DistinguishedFolderId, либо FolderId (EWS-REFERENCE.md §4–§5, §8).
type parentFolderIDsRequest struct {
	XMLName               xml.Name                      `xml:"http://schemas.microsoft.com/exchange/services/2006/messages ParentFolderIds"`
	DistinguishedFolderID *distinguishedFolderIDRequest `xml:",omitempty"`
	FolderID              *folderIDRequest              `xml:",omitempty"`
}

// distinguishedFolderIDRequest — t:DistinguishedFolderId@Id (msgfolderroot,
// inbox, calendar, ... — EWS-REFERENCE.md §9).
type distinguishedFolderIDRequest struct {
	XMLName xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/types DistinguishedFolderId"`
	ID      string   `xml:"Id,attr"`
}

// folderIDRequest — t:FolderId@Id,@ChangeKey для обычной папки из FindFolder.
type folderIDRequest struct {
	XMLName   xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/types FolderId"`
	ID        string   `xml:"Id,attr"`
	ChangeKey string   `xml:"ChangeKey,attr,omitempty"`
}

// responseHeader — общая шапка m:*ResponseMessage: ResponseClass="Error"
// превращается в *ResponseError (EWS-REFERENCE.md §9).
type responseHeader struct {
	ResponseClass string `xml:"ResponseClass,attr"`
	ResponseCode  string `xml:"ResponseCode"`
	MessageText   string `xml:"MessageText"`
}

// err превращает ResponseClass="Error" в *ResponseError (Success/Warning — nil).
func (h responseHeader) err() error {
	return checkResponseClass(h.ResponseClass, h.ResponseCode, h.MessageText)
}

// findFolderResponse — m:FindFolderResponse (разбор по локальным именам:
// префиксы реального сервера могут отличаться, см. soap.go parseResponse).
type findFolderResponse struct {
	XMLName          xml.Name `xml:"FindFolderResponse"`
	ResponseMessages struct {
		FindFolderResponseMessage findFolderResponseMessage `xml:"FindFolderResponseMessage"`
	} `xml:"ResponseMessages"`
}

type findFolderResponseMessage struct {
	responseHeader
	RootFolder struct {
		TotalNumberOfFoldersInView int `xml:"TotalNumberOfFoldersInView,attr"`
		Folders                    struct {
			Folder []xmlFolder `xml:"Folder"`
		} `xml:"Folders"`
	} `xml:"RootFolder"`
}

// xmlFolder — t:Folder из t:Folders.
type xmlFolder struct {
	FolderID         xmlFolderID `xml:"FolderId"`
	DisplayName      string      `xml:"DisplayName"`
	TotalCount       int         `xml:"TotalCount"`
	UnreadCount      int         `xml:"UnreadCount"`
	ChildFolderCount int         `xml:"ChildFolderCount"`
}

type xmlFolderID struct {
	ID        string `xml:"Id,attr"`
	ChangeKey string `xml:"ChangeKey,attr"`
}

// result проверяет ResponseClass и конвертирует ответ в []Folder.
func (r *findFolderResponse) result() ([]Folder, error) {
	msg := &r.ResponseMessages.FindFolderResponseMessage
	if err := msg.err(); err != nil {
		return nil, err
	}
	folders := make([]Folder, 0, len(msg.RootFolder.Folders.Folder))
	for _, f := range msg.RootFolder.Folders.Folder {
		folders = append(folders, Folder{
			ID:               f.FolderID.ID,
			ChangeKey:        f.FolderID.ChangeKey,
			DisplayName:      f.DisplayName,
			TotalCount:       f.TotalCount,
			UnreadCount:      f.UnreadCount,
			ChildFolderCount: f.ChildFolderCount,
		})
	}
	return folders, nil
}
