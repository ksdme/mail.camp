package models

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"strings"

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
	wildcardPrefixMaxSize    = 24
	randomMailboxNameMinSize = 32
)

// TODO: Add created at, updated at times.
type Account struct {
	ID            int64          `bun:",pk,autoincrement"`
	KeySignature  string         `bun:",notnull,unique"`
	MailboxPrefix sql.NullString `bun:",unique"`
}

func (account *Account) createMailbox(ctx context.Context, db *bun.DB, name string) (*Mailbox, error) {
	name = strings.ToLower(name)
	name = strings.TrimSpace(name)
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
			// TODO: Add the fact that the names cannot start with a symbol either.
			"invalid name, a name can only contain lower case letters, numbers, periods, underscore or hyphens",
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
func (account *Account) CreateRandomMailbox(ctx context.Context, db *bun.DB) (*Mailbox, error) {
	var name string
	// Keep looping until we find a free name.
	// TODO: We should cap this.
	for {
		name = ""
		for len(name) < randomMailboxNameMinSize {
			index := rand.Intn(len(wordlist.Words))
			name += wordlist.Words[index]
		}

		if exists, err := db.NewSelect().Where("name = ?", name).Exists(ctx); err != nil {
			return nil, errors.Wrap(err, "error while finding an unused name")
		} else if !exists {
			break
		}
	}

	return account.createMailbox(ctx, db, name)
}

// Generate a mailbox based on the configured prefix on the account.
func (account *Account) CreateWildcardMailbox(ctx context.Context, db *bun.DB, suffix string) (*Mailbox, error) {
	if !account.MailboxPrefix.Valid {
		return nil, fmt.Errorf("no mailbox prefix configured for the account")
	}

	if suffix == "" {
		return nil, errors.Wrap(ErrInvalidMailbox, "cannot have an empty suffix")
	}

	return account.createMailbox(
		ctx,
		db,
		fmt.Sprintf("%s.%s", account.MailboxPrefix.String, suffix),
	)
}
