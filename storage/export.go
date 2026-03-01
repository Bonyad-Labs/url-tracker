package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func (s *Store) ImportBookmarks(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return s.ImportBookmarksFromReader(file)
}

// ImportBookmarksFromReader reads from an io.Reader, allowing for tests without files
func (s *Store) ImportBookmarksFromReader(r io.Reader) error {
	doc, err := html.Parse(r)
	if err != nil {
		return err
	}

	var parseNode func(*html.Node, string)
	parseNode = func(n *html.Node, folder string) {
		// Netscape Bookmark format: Folders are marked by <H3> tags.
		// When we find an <H3>, we update the folder name for subsequent nodes in this branch.
		if n.Type == html.ElementNode && n.Data == "h3" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				folder = n.FirstChild.Data
			}
		}

		// When we find an <A> tag, it's a bookmark. We use the most recent folder name found in this scope.
		if n.Type == html.ElementNode && n.Data == "a" {
			var href, addDate string
			if idx := slices.IndexFunc(n.Attr, func(a html.Attribute) bool { return a.Key == "href" }); idx != -1 {
				href = n.Attr[idx].Val
			}
			if idx := slices.IndexFunc(n.Attr, func(a html.Attribute) bool { return a.Key == "add_date" }); idx != -1 {
				addDate = n.Attr[idx].Val
			}

			if href != "" {
				title := ""
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					title = n.FirstChild.Data
				}

				timestamp := time.Now().Unix()
				if addDate != "" {
					if ts, err := strconv.ParseInt(addDate, 10, 64); err == nil {
						timestamp = ts
					}
				}

				// Only import if it's a valid remote URL
				if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
					entry := Entry{
						URL:       href,
						Title:     title,
						Category:  folder,
						Timestamp: timestamp,
					}

					// Ignore err for simple duplicates
					_ = s.AddEntry(entry)
				}
			}
		}

		// Traverse children. We pass the folder name into the recursive call.
		// We avoid using a shared scoped variable because sibling nodes in Netscape HTML
		// (like the folder title H3 and the DL containing its contents) need to share
		// the same folder context without one sibling's metadata leaking or resetting improperly.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			// If we just encountered a folder title sibling, we want subsequent siblings
			// (like the DL list) to inherit it.
			if n.Type == html.ElementNode && n.Data == "h3" {
				// Internal nodes of H3 are just text, no need to update folder for them
				parseNode(c, folder)
			} else {
				// For most nodes (like DT), if a child sets a new folder name,
				// we need to see it here so we can pass it to the NEXT sibling.
				// This is why we update the local 'folder' variable.
				if c.Type == html.ElementNode && c.Data == "h3" && c.FirstChild != nil {
					folder = c.FirstChild.Data
				}
				parseNode(c, folder)
			}
		}
	}

	parseNode(doc, "")
	return nil
}

func (s *Store) ExportBookmarks(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return s.ExportBookmarksToWriter(file)
}

// ExportBookmarksToWriter writes to an io.Writer, allowing for tests without files
func (s *Store) ExportBookmarksToWriter(w io.Writer) error {
	entries := s.GetEntries()

	// Group entries by Category
	groups := make(map[string][]Entry)
	for _, e := range entries {
		cat := strings.TrimSpace(e.Category)
		if cat == "" {
			cat = "Uncategorized"
		}
		groups[cat] = append(groups[cat], e)
	}

	// Write Netscape Bookmark Header
	header := `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- This is an automatically generated file.
     It will be read and overwritten.
     DO NOT EDIT! -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
`
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}

	for _, category := range slices.Sorted(maps.Keys(groups)) {
		items := groups[category]
		_, err := fmt.Fprintf(w, "    <DT><H3 ADD_DATE=\"%d\" LAST_MODIFIED=\"%d\">%s</H3>\n    <DL><p>\n", time.Now().Unix(), time.Now().Unix(), html.EscapeString(category))
		if err != nil {
			return err
		}

		for _, item := range items {
			_, err = fmt.Fprintf(w, "        <DT><A HREF=\"%s\" ADD_DATE=\"%d\">%s</A>\n", html.EscapeString(item.URL), item.Timestamp, html.EscapeString(item.Title))
			if err != nil {
				return err
			}
		}

		if _, err = io.WriteString(w, "    </DL><p>\n"); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, "</DL><p>\n")
	return err
}

func (s *Store) ExportNativeJSON(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return s.ExportNativeJSONToWriter(file)
}

// ExportNativeJSONToWriter writes to an io.Writer
func (s *Store) ExportNativeJSONToWriter(w io.Writer) error {
	entries := s.GetEntries()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func (s *Store) ImportNativeJSON(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return s.ImportNativeJSONFromReader(file)
}

// ImportNativeJSONFromReader reads from an io.Reader
func (s *Store) ImportNativeJSONFromReader(r io.Reader) error {
	var entries []Entry
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&entries); err != nil {
		return err
	}

	for _, entry := range entries {
		// Ignore duplicates
		_ = s.AddEntry(entry)
	}

	return nil
}
