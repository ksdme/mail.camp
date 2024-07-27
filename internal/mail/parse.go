package mail

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"

	"github.com/k3a/html2text"
	"github.com/pkg/errors"
)

// How do we select the relevant message?
//
// 1. If the body does not contain multiple parts, the body is returned.
// 2. We iterate on the parts,
//  1. If the part itself is an multipart, start iterating on those parts,
//  2. Ignore the part if it is not a text, html part
//  3. If the part has a Content Disposition of attachment, ignore
//  4. If the part is a text part, return it
//  5. Track this part if we haven't already seen an html part already
//
// 3. If we have a tracked html part, text-ify and return it
// 4. Return a string saying "empty body"
func extractPlainText(message *mail.Message) (string, error) {
	readAll := func(reader io.Reader) (string, error) {
		if value, err := io.ReadAll(reader); err != nil {
			return "", errors.Wrap(err, "could not continue reading body")
		} else {
			return string(value), nil
		}
	}

	var text string
	var html string
	var resolve func(io.Reader, string, string) error
	resolve = func(r io.Reader, cType string, cDisposition string) error {
		// Handles the content type is not available at all.
		if cType == "" {
			if value, err := readAll(r); err != nil {
				return err
			} else {
				text = value
				return nil
			}
		}

		// We hate attachments.
		if cDisposition != "" {
			label, params, err := mime.ParseMediaType(cDisposition)
			if err != nil {
				return errors.Wrap(
					err,
					fmt.Sprintf("could not parse disposition value (%s)", cDisposition),
				)
			}
			if label == "attachment" {
				return nil
			}
			if _, ok := params["filename"]; ok {
				return nil
			}
		}

		mediaType, params, err := mime.ParseMediaType(cType)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not parse media type from %s", cType))
		}

		// Handle the nested-ness of the parse.
		if strings.HasPrefix(mediaType, "multipart/") {
			boundary, ok := params["boundary"]
			if !ok {
				return fmt.Errorf("unknown multipart boundary")
			}

			parts := multipart.NewReader(r, boundary)
			for {
				part, err := parts.NextPart()
				if err == io.EOF {
					return nil
				} else if err != nil {
					return err
				}

				err = resolve(
					part,
					part.Header.Get("Content-Type"),
					part.Header.Get("Content-Disposition"),
				)
				if err != nil {
					return err
				}
				if len(text) > 0 {
					return nil
				}
			}
		}

		// Handle leaf parts.
		switch mediaType {
		case "":
		case "text/plain":
			if value, err := readAll(r); err != nil {
				return err
			} else {
				text = value
				return nil
			}

		case "text/html":
			// Reading this is not a problem becase
			if len(html) == 0 {
				if value, err := readAll(r); err != nil {
					return err
				} else {
					html = value
					return nil
				}
			}
		}

		return nil
	}

	err := resolve(
		message.Body,
		message.Header.Get("Content-Type"),
		message.Header.Get("Content-Disposition"),
	)
	if err != nil {
		return "", nil
	}

	if len(text) > 0 {
		return text, nil
	}

	if len(html) > 0 {
		value := html2text.HTML2TextWithOptions(
			html,
			html2text.WithLinksInnerText(),
			html2text.WithListSupport(),
		)
		return value, nil
	}

	return "empty message :(", nil
}
