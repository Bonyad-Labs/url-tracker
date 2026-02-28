package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ImportBookmarks reads a Netscape Bookmark HTML file and inserts the entries into the database.
func (s *Store) ImportBookmarks(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	doc, err := html.Parse(file)
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
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
				} else if a.Key == "add_date" {
					addDate = a.Val
				}
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

// ExportBookmarks writes all entries to a Netscape Bookmark HTML file.
func (s *Store) ExportBookmarks(filepath string) error {
	entries := s.GetEntries()
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

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
	_, err = io.WriteString(file, header)
	if err != nil {
		return err
	}

	// Group entries by Category
	groups := make(map[string][]Entry)
	for _, e := range entries {
		cat := strings.TrimSpace(e.Category)
		if cat == "" {
			cat = "Uncategorized"
		}
		groups[cat] = append(groups[cat], e)
	}

	for category, items := range groups {
		_, err = fmt.Fprintf(file, "    <DT><H3 ADD_DATE=\"%d\" LAST_MODIFIED=\"%d\">%s</H3>\n    <DL><p>\n", time.Now().Unix(), time.Now().Unix(), html.EscapeString(category))
		if err != nil {
			return err
		}

		for _, item := range items {
			_, err = fmt.Fprintf(file, "        <DT><A HREF=\"%s\" ADD_DATE=\"%d\">%s</A>\n", html.EscapeString(item.URL), item.Timestamp, html.EscapeString(item.Title))
			if err != nil {
				return err
			}
		}

		_, err = io.WriteString(file, "    </DL><p>\n")
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(file, "</DL><p>\n")
	return err
}

// ExportNativeJSON writes all entries to a native `.json` backup file.
func (s *Store) ExportNativeJSON(filepath string) error {
	entries := s.GetEntries()
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

// ImportNativeJSON reads a native `.json` backup file and imports the entries.
func (s *Store) ImportNativeJSON(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	var entries []Entry
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&entries); err != nil {
		return err
	}

	for _, entry := range entries {
		// Ignore duplicates
		_ = s.AddEntry(entry)
	}

	return nil
}
