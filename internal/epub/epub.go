package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
	"time"
)

type EPUBWriter struct {
	zipWriter *zip.Writer
	pages     []string
	images    []string
	title     string
	pageCount int
}

func NewEPUBWriter(writer io.Writer, title string) *EPUBWriter {
	return &EPUBWriter{
		zipWriter: zip.NewWriter(writer),
		title:     title,
		pages:     make([]string, 0),
		images:    make([]string, 0),
		pageCount: 0,
	}
}

func (e *EPUBWriter) Close() error {
	// Write the EPUB structure files
	if err := e.writeMimeType(); err != nil {
		return err
	}

	if err := e.writeContainer(); err != nil {
		return err
	}

	if err := e.writeOPF(); err != nil {
		return err
	}

	if err := e.writeNCX(); err != nil {
		return err
	}

	return e.zipWriter.Close()
}

func (e *EPUBWriter) AddPage(filename string, data []byte) error {
	// Add image to EPUB
	imageFile, err := e.zipWriter.Create(fmt.Sprintf("OEBPS/images/%s", filename))
	if err != nil {
		return err
	}

	if _, err := imageFile.Write(data); err != nil {
		return err
	}

	// Create XHTML page for this image
	pageNum := e.pageCount + 1
	xhtmlFilename := fmt.Sprintf("page%d.xhtml", pageNum)

	xhtmlFile, err := e.zipWriter.Create(fmt.Sprintf("OEBPS/%s", xhtmlFilename))
	if err != nil {
		return err
	}

	xhtmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <title>Page %d</title>
    <style type="text/css">
        html, body {
            margin: 0;
            padding: 0;
            height: 100%%;
            width: 100%%;
            overflow: hidden;
        }
        .page-container {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            width: 100vw;
            background-color: #ffffff;
        }
        .page-image {
            max-width: 100%%;
            max-height: 100%%;
            width: auto;
            height: auto;
            object-fit: contain;
            display: block;
        }
        /* Fallback for older e-readers */
        body {
            text-align: center;
            margin: 0;
            padding: 0;
        }
        img {
            max-width: 100%%;
            max-height: 100%%;
            width: auto;
            height: auto;
        }
    </style>
</head>
<body>
    <div class="page-container">
        <img class="page-image" src="images/%s" alt="Page %d"/>
    </div>
</body>
</html>`, pageNum, filename, pageNum)

	if _, err := xhtmlFile.Write([]byte(xhtmlContent)); err != nil {
		return err
	}

	e.pages = append(e.pages, xhtmlFilename)
	e.images = append(e.images, filename)
	e.pageCount++

	return nil
}

func (e *EPUBWriter) writeMimeType() error {
	file, err := e.zipWriter.Create("mimetype")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte("application/epub+zip"))
	return err
}

func (e *EPUBWriter) writeContainer() error {
	file, err := e.zipWriter.Create("META-INF/container.xml")
	if err != nil {
		return err
	}

	content := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
    <rootfiles>
        <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
    </rootfiles>
</container>`

	_, err = file.Write([]byte(content))
	return err
}

func (e *EPUBWriter) writeOPF() error {
	file, err := e.zipWriter.Create("OEBPS/content.opf")
	if err != nil {
		return err
	}

	var manifestItems strings.Builder
	var spineItems strings.Builder

	for i, page := range e.pages {
		pageId := fmt.Sprintf("page%d", i+1)
		imageId := fmt.Sprintf("img%d", i+1)

		manifestItems.WriteString(fmt.Sprintf(`        <item id="%s" href="%s" media-type="application/xhtml+xml"/>
`, pageId, page))
		manifestItems.WriteString(fmt.Sprintf(`        <item id="%s" href="images/%s" media-type="image/jpeg"/>
`, imageId, e.images[i]))

		spineItems.WriteString(fmt.Sprintf(`        <itemref idref="%s"/>
`, pageId))
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="book-id">
    <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
        <dc:title>%s</dc:title>
        <dc:language>en</dc:language>
        <dc:identifier id="book-id">%s</dc:identifier>
        <dc:creator>Comic Downloader</dc:creator>
        <dc:date>%s</dc:date>
        <meta name="cover" content="img1"/>
    </metadata>
    <manifest>
        <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
%s    </manifest>
    <spine toc="ncx">
%s    </spine>
</package>`, e.title, e.title, time.Now().Format("2006-01-02"), manifestItems.String(), spineItems.String())

	_, err = file.Write([]byte(content))
	return err
}

func (e *EPUBWriter) writeNCX() error {
	file, err := e.zipWriter.Create("OEBPS/toc.ncx")
	if err != nil {
		return err
	}

	var navPoints strings.Builder
	for i, page := range e.pages {
		navPoints.WriteString(fmt.Sprintf(`        <navPoint id="page%d" playOrder="%d">
            <navLabel>
                <text>Page %d</text>
            </navLabel>
            <content src="%s"/>
        </navPoint>
`, i+1, i+1, i+1, page))
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ncx version="2005-1" xmlns="http://www.daisy.org/z3986/2005/ncx/">
    <head>
        <meta name="dtb:uid" content="%s"/>
        <meta name="dtb:depth" content="1"/>
        <meta name="dtb:totalPageCount" content="%d"/>
        <meta name="dtb:maxPageNumber" content="%d"/>
    </head>
    <docTitle>
        <text>%s</text>
    </docTitle>
    <navMap>
%s    </navMap>
</ncx>`, e.title, e.pageCount, e.pageCount, e.title, navPoints.String())

	_, err = file.Write([]byte(content))
	return err
}
