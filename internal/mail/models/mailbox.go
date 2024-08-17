package models

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"strings"

	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/utils"
	"github.com/ksdme/mail/internal/wordlist"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidMailbox = errors.New("invalid mailbox")
)

// Because we support both wildcard mailboxes based on the prefix
// on your account and generating random mailboxes, there could be
// an issue where the random mailbox name collides with the wildcard
// mailbox name. To prevent this, we have a strict shape for the name
// of each type of mailbox.
// Wildcard mailboxes can have a purely alpha numeric prefix of maximum length x.
// Then, the prefix and suffix will be separated using a period.
// On the other hand, random mailboxes will be purely alphabetic and have
// a minimum length that is greater than x.
const (
	wildcardPrefixMaxSize    = 16
	randomMailboxNameMinSize = 18
)

// TODO: Add CreatedAt, UpdatedAt fields.
type Mailbox struct {
	ID   int64  `bun:",pk,autoincrement"`
	Name string `bun:",notnull"`

	// TODO: We need to setup cascade relationship.
	AccountID int64           `bun:",notnull"`
	Account   *models.Account `bun:"rel:belongs-to,join:account_id=id"`
}

func (m Mailbox) Email() string {
	return fmt.Sprintf("%s@%s", m.Name, config.Mail.MXHost)
}

func createMailbox(ctx context.Context, db *bun.DB, account models.Account, name string) (*Mailbox, error) {
	name = normalizeMailbox(name)
	if len(name) <= 2 {
		return nil, errors.Wrap(
			ErrInvalidMailbox,
			"name is too short, it needs to be longer than 2 characters",
		)
	}

	// Match the valid character set.
	// If the pattern below is updated, it needs to be reflected in the check below too.
	pattern := `^[a-z\d][a-z\d\.\-\_\+]+[a-z\d]$`
	if matched, err := regexp.MatchString(pattern, name); err != nil {
		return nil, errors.Wrap(err, "could not validate mailbox name")
	} else if !matched {
		return nil, errors.Wrap(
			ErrInvalidMailbox,
			// TODO: Maybe break this down into multiple checks.
			"invalid name, a name can only contain lower case letters, numbers, periods, "+
				"underscores or hyphens, and, it needs to being and end with an alphanumeric character",
		)
	}

	// Check if name contains repeating symbols.
	for _, character := range []string{".", "-", "_"} {
		if strings.Contains(name, character+character) {
			return nil, errors.Wrap(
				ErrInvalidMailbox,
				"invalid name, a name cannot contain consecutive special symbols (periods, underscores or hyphens)",
			)
		}
	}

	// While the standard says that this limit is 64, we don't really care,
	// but we cannot allow for an infinite size either.
	if len(name) > 128 {
		return nil, errors.Wrap(
			ErrInvalidMailbox,
			"a name cannot be longer than 128 characters",
		)
	}

	// Create the mailbox while checking for duplicate.
	mailbox := &Mailbox{Name: name, AccountID: account.ID}
	if _, err := db.NewInsert().Model(mailbox).Exec(ctx); err != nil {
		if utils.IsUniqueConstraintErr(err) {
			return nil, errors.Wrap(
				ErrInvalidMailbox,
				"a mailbox with this name already exists",
			)
		}

		slog.Error("unknown error while creating the mailbox", "error", err)
		return nil, errors.Wrap(
			ErrInvalidMailbox,
			"unknown error occurred while creating the mailbox",
		)
	}

	return mailbox, nil
}

// Create a mailbox against this account with a random name.
func CreateRandomMailbox(ctx context.Context, db *bun.DB, account models.Account) (*Mailbox, error) {
	var name string
	// Keep looping until we find a free name.
	// TODO: We should cap this.
	for {
		name = ""
		for len(name) < randomMailboxNameMinSize {
			index := rand.Intn(len(wordlist.Words))
			name += wordlist.Words[index]
		}
		name = normalizeMailbox(name)

		if exists, err := db.NewSelect().Model(&Mailbox{}).Where("name = ?", name).Exists(ctx); err != nil {
			return nil, errors.Wrap(err, "error while finding an unused name")
		} else if !exists {
			break
		}
	}

	return createMailbox(ctx, db, account, name)
}

// Generate a mailbox based on the configured prefix on the account.
func CreateWildcardMailbox(ctx context.Context, db *bun.DB, account models.Account, suffix string) (*Mailbox, error) {
	if !account.ReservedPrefix.Valid {
		return nil, fmt.Errorf("no mailbox prefix configured for the account")
	}

	if suffix == "" {
		return nil, errors.Wrap(ErrInvalidMailbox, "cannot have an empty suffix")
	}

	return createMailbox(
		ctx,
		db,
		account,
		fmt.Sprintf("%s.%s", account.ReservedPrefix.String, suffix),
	)
}

// Finds an existing mailbox with a name or creates one if necessary or possible.
func GetOrCreateMailbox(ctx context.Context, db *bun.DB, name string) (*Mailbox, error) {
	name = normalizeMailbox(name)

	// Try finding an existing mailbox.
	mailbox := &Mailbox{}
	if err := db.NewSelect().Model(mailbox).Where("name = ?", name).Scan(ctx); err != nil {
		if err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "could not query mailboxes")
		}
	} else {
		return mailbox, nil
	}

	// If the name is wildcard compatible, try to issue a mailbox.
	if strings.Contains(name, ".") {
		sections := strings.SplitN(name, ".", 2)

		account := models.Account{}
		if err := db.NewSelect().Model(&account).Where("mailbox_prefix = ?", sections[0]).Scan((ctx)); err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("unknown mailbox prefix")
			}

			return nil, errors.Wrap(err, "could not query mailboxes for wildcards")
		}

		return CreateWildcardMailbox(ctx, db, account, sections[1])
	}

	return nil, fmt.Errorf("could not find or create mailbox")
}

// Normalizes the mailbox name.
// TODO: It should also deal with unicode characters.
func normalizeMailbox(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return name
}
