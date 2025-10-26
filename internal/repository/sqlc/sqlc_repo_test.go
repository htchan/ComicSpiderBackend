package sqlc

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestNewRepo(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	tests := []struct {
		name string
		db   *sql.DB
	}{
		{
			name: "providing a sqlc database",
			db:   db,
		},
		{
			name: "providing nil database",
			db:   nil,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repo := NewRepo(test.db, &config.WebsiteConfig{})
			if test.db != nil {
				assert.Equal(t, test.db.Stats(), repo.stats())
			}
		})
	}
}

func TestSqlcRepo_CreateWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})
	t.Cleanup(func() {
		db.Exec("delete from websites where title=$1", "unknown")
		db.Close()
	})

	uuid := "create-website-uuid"
	userUUID := "create-website-user-uuid"
	title := "create website"
	populateData(db, uuid, title, userUUID, "active")

	tests := []struct {
		name        string
		web         model.Website
		expect      model.Website
		expectError error
	}{
		{
			name: "create a new website",
			web: model.Website{
				UUID:       "dcb12928-5b5b-43f3-9d0e-ddb526d9794d",
				URL:        "http://example.com",
				Title:      "unknown",
				RawContent: "",
				UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expect: model.Website{
				UUID:       "dcb12928-5b5b-43f3-9d0e-ddb526d9794d",
				URL:        "http://example.com",
				Title:      "unknown",
				RawContent: "",
				UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{},
			},
			expectError: nil,
		},
		{
			name: "create an existing website",
			web: model.Website{
				UUID:       uuid,
				URL:        "http://example.com/" + title,
				Title:      title,
				RawContent: "",
				UpdateTime: time.Now().UTC().Truncate(MinTimeUnit),
			},
			expect: model.Website{
				UUID:       uuid,
				URL:        "http://example.com/" + title,
				Title:      title,
				RawContent: "content",
				UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Conf:       &config.WebsiteConfig{},
			},
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := r.CreateWebsite(context.Background(), &test.web)
			assert.ErrorIs(t, err, test.expectError)
			assert.Equal(t, test.expect, test.web)
		})
	}
}

func TestSqlcRepo_UpdateWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "update-website-uuid"
	userUUID := "update-website-user-uuid"
	title := "update website"
	populateData(db, uuid, title, userUUID, "active")

	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", title)
		db.Close()
	})

	tests := []struct {
		name        string
		web         model.Website
		expect      *model.Website
		expectError error
	}{
		{
			name: "update successfully",
			web: model.Website{
				UUID:       uuid,
				URL:        "http://example.com/" + title,
				Title:      title,
				RawContent: "content new",
				UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expect: &model.Website{
				UUID:       uuid,
				URL:        "http://example.com/" + title,
				Title:      title,
				RawContent: "content new",
				UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Status:     "active",
				Conf:       &config.WebsiteConfig{},
			},
			expectError: nil,
		},
		{
			name: "update not exist website",
			web: model.Website{
				UUID:       "uuid-that-not-exist",
				URL:        "http://example.com/not-exist",
				Title:      title,
				RawContent: "",
				UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := r.UpdateWebsite(context.Background(), &test.web)
			assert.ErrorIs(t, err, test.expectError)

			web, _ := r.FindWebsite(context.Background(), test.web.UUID)
			assert.Equal(t, test.expect, web)
		})
	}
}

func TestSqlcRepo_DeleteWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "delete-website-uuid"
	userUUID := "delete-website-user-uuid"
	title := "delete website"
	populateData(db, uuid, title, userUUID, "active")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		webUUID     string
		expectError error
	}{
		{
			name:        "delete successfully",
			webUUID:     uuid,
			expectError: nil,
		},
		{
			name:        "delete not exist",
			webUUID:     "uuid-that-not-exist",
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := r.DeleteWebsite(context.Background(), &model.Website{UUID: test.webUUID})
			assert.ErrorIs(t, err, test.expectError)

			web, err := r.FindWebsite(context.Background(), test.webUUID)
			assert.ErrorIs(t, err, sql.ErrNoRows)
			assert.Nil(t, web)
		})
	}
}

func TestSqlcRepo_FindWebsites(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "find-websites-uuid"
	userUUID := "find-websites-user-uuid"
	title := "find websites"
	populateData(db, uuid, title, userUUID, "active")
	uuidReadOnly := "find-websites-readonly-uuid"
	populateData(db, uuidReadOnly, title+"-readonly", userUUID, "read_only")
	uuidInactive := "find-websites-inactive-uuid"
	populateData(db, uuidInactive, title+"-inactive", userUUID, "inactive")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from websites where uuid=$1", uuidReadOnly)
		db.Exec("delete from websites where uuid=$1", uuidInactive)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name          string
		expectInclude []model.Website
		expectExclude []model.Website
		expectError   error
	}{
		{
			name: "happy flow",
			expectInclude: []model.Website{
				{
					UUID:       uuid,
					URL:        "http://example.com/" + title,
					Title:      title,
					RawContent: "content",
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Status:     "active",
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectExclude: []model.Website{
				{
					UUID:       uuidReadOnly,
					URL:        "http://example.com/" + title + "-readonly",
					Title:      title + "-readonly",
					RawContent: "content",
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Status:     "read_only",
					Conf:       &config.WebsiteConfig{},
				},
				{
					UUID:       uuidInactive,
					URL:        "http://example.com/" + title + "-inactive",
					Title:      title + "-inactive",
					RawContent: "content",
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Status:     "inactive",
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := r.FindWebsites(context.Background())
			assert.ErrorIs(t, err, test.expectError)
			for _, expect := range test.expectInclude {
				assert.Contains(t, result, expect)
			}
			for _, exclude := range test.expectExclude {
				assert.NotContains(t, result, exclude)
			}
		})
	}
}

func TestSqlcRepo_FindWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "find-website-uuid"
	userUUID := "find-website-user-uuid"
	title := "find website"
	populateData(db, uuid, title, userUUID, "active")
	uuidReadOnly := "find-website-readonly-uuid"
	populateData(db, uuidReadOnly, title+"-readonly", userUUID, "read_only")
	uuidInactive := "find-website-inactive-uuid"
	populateData(db, uuidInactive, title+"-inactive", userUUID, "inactive")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from websites where uuid=$1", uuidReadOnly)
		db.Exec("delete from websites where uuid=$1", uuidInactive)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		webUUID     string
		expect      *model.Website
		expectError error
	}{
		{
			name:    "find active website",
			webUUID: uuid,
			expect: &model.Website{
				UUID:       uuid,
				URL:        "http://example.com/" + title,
				Title:      title,
				RawContent: "content",
				UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Status:     "active",
				Conf:       &config.WebsiteConfig{},
			},
			expectError: nil,
		},
		{
			name:    "find read-only website",
			webUUID: uuidReadOnly,
			expect: &model.Website{
				UUID:       uuidReadOnly,
				URL:        "http://example.com/" + title + "-readonly",
				Title:      title + "-readonly",
				RawContent: "content",
				UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Status:     "read_only",
				Conf:       &config.WebsiteConfig{},
			},
		},
		{
			name:        "find inactive website",
			webUUID:     uuidInactive,
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
		{
			name:        "find not exist website",
			webUUID:     "uuid-that-not-exist",
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := r.FindWebsite(context.Background(), test.webUUID)
			assert.ErrorIs(t, err, test.expectError)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestSqlcRepo_CreateUserWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "create-user-website-uuid"
	userUUID := "create-user-website-user-uuid"
	title := "create user website"
	populateData(db, uuid, title, userUUID, "active")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		web         model.UserWebsite
		expect      model.UserWebsite
		expectError error
	}{
		{
			name: "create new user website",
			web: model.UserWebsite{
				WebsiteUUID: uuid,
				UserUUID:    "other-user-uuid",
				GroupName:   "title",
				AccessTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expect: model.UserWebsite{
				WebsiteUUID: uuid,
				UserUUID:    "other-user-uuid",
				GroupName:   "title",
				AccessTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Website: model.Website{
					UUID:       uuid,
					URL:        "http://example.com/" + title,
					Title:      title,
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					RawContent: "content",
					Status:     "active",
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectError: nil,
		},
		{
			name: "create existing user website",
			web: model.UserWebsite{
				WebsiteUUID: uuid,
				UserUUID:    userUUID,
			},
			expect: model.UserWebsite{
				WebsiteUUID: uuid,
				UserUUID:    userUUID,
				GroupName:   title,
				AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Website: model.Website{
					UUID:       uuid,
					URL:        "http://example.com/" + title,
					Title:      title,
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					RawContent: "content",
					Status:     "active",
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectError: nil,
		},
		{
			name: "create new user website link to not exist website",
			web: model.UserWebsite{
				WebsiteUUID: "not-exist-uuid",
				UserUUID:    "new",
				GroupName:   "title",
				AccessTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expect: model.UserWebsite{
				WebsiteUUID: "not-exist-uuid",
				UserUUID:    "new",
				GroupName:   "title",
				AccessTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectError: sql.ErrNoRows,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := r.CreateUserWebsite(context.Background(), &test.web)
			assert.ErrorIs(t, err, test.expectError)
			assert.Equal(t, test.expect, test.web)
		})
	}
}

func TestSqlcRepo_UpdateUserWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "update-user-website-uuid"
	userUUID := "update-user-website-user-uuid"
	title := "update user website"
	populateData(db, uuid, title, userUUID, "active")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		web         model.UserWebsite
		expect      *model.UserWebsite
		expectError error
	}{
		{
			name: "update existing website",
			web: model.UserWebsite{
				WebsiteUUID: uuid,
				UserUUID:    userUUID,
				GroupName:   title,
				AccessTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expect: &model.UserWebsite{
				WebsiteUUID: uuid,
				UserUUID:    userUUID,
				GroupName:   title,
				AccessTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				Website: model.Website{
					UUID:       uuid,
					URL:        "http://example.com/" + title,
					Title:      title,
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectError: nil,
		},
		{
			name: "update not exist user website",
			web: model.UserWebsite{
				WebsiteUUID: "not-exist-website-uuid",
				UserUUID:    "not-exist-user-uuid",
			},
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := r.UpdateUserWebsite(context.Background(), &test.web)
			assert.ErrorIs(t, err, test.expectError)

			web, _ := r.FindUserWebsite(context.Background(), test.web.UserUUID, test.web.WebsiteUUID)
			assert.Equal(t, test.expect, web)
		})
	}
}

func TestSqlcRepo_DeleteUserWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "delete-user-website-uuid"
	userUUID := "delete-user-website-user-uuid"
	title := "delete user website"
	populateData(db, uuid, title, userUUID, "active")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		userUUID    string
		webUUID     string
		expectError error
	}{
		{
			name:        "delete successfully",
			webUUID:     uuid,
			userUUID:    userUUID,
			expectError: nil,
		},
		{
			name:        "delete not exist",
			webUUID:     "not exist",
			userUUID:    "not exist",
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := r.DeleteUserWebsite(context.Background(), &model.UserWebsite{
				UserUUID:    test.userUUID,
				WebsiteUUID: test.webUUID,
			})
			assert.ErrorIs(t, err, test.expectError)

			web, err := r.FindUserWebsite(context.Background(), test.userUUID, test.webUUID)
			assert.ErrorIs(t, err, sql.ErrNoRows)
			assert.Nil(t, web)
		})
	}
}

func TestSqlcRepo_FindUserWebsites(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "find-user-websites-uuid"
	userUUID := "find-user-websites-user-uuid"
	title := "find user websites"
	populateData(db, uuid, title, userUUID, "active")
	uuidReadOnly := "find-user-websites-readonly-uuid"
	populateData(db, uuidReadOnly, title+"-readonly", userUUID, "read_only")
	uuidInactive := "find-user-websites-inactive-uuid"
	populateData(db, uuidInactive, title+"-inactive", userUUID, "inactive")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from websites where uuid=$1", uuidReadOnly)
		db.Exec("delete from websites where uuid=$1", uuidInactive)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		userUUID    string
		expect      model.UserWebsites
		expectError error
	}{
		{
			name:     "find web of existing user",
			userUUID: userUUID,
			expect: model.UserWebsites{
				{
					UserUUID:    userUUID,
					WebsiteUUID: uuidReadOnly,
					GroupName:   title + "-readonly",
					AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Website: model.Website{
						UUID:       uuidReadOnly,
						URL:        "http://example.com/" + title + "-readonly",
						Title:      title + "-readonly",
						UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
						Conf:       &config.WebsiteConfig{},
					},
				},
				{
					UserUUID:    userUUID,
					WebsiteUUID: uuid,
					GroupName:   title,
					AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Website: model.Website{
						UUID:       uuid,
						URL:        "http://example.com/" + title,
						Title:      title,
						UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
						Conf:       &config.WebsiteConfig{},
					},
				},
			},
			expectError: nil,
		},
		{
			name:        "find web of not existing user",
			userUUID:    "not exist",
			expect:      model.UserWebsites{},
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := r.FindUserWebsites(context.Background(), test.userUUID)
			assert.ErrorIs(t, err, test.expectError)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestSqlcRepo_FindUserWebsitesByGroup(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "find-user-websites-group-uuid"
	userUUID := "find-user-websites-group-user-uuid"
	title := "find user websites group"
	populateData(db, uuid, title, userUUID, "active")
	uuidReadOnly := "find-user-websites-group-readonly-uuid"
	populateData(db, uuidReadOnly, title+"-readonly", userUUID, "read_only")
	uuidInactive := "find-user-websites-group-inactive-uuid"
	populateData(db, uuidInactive, title+"-inactive", userUUID, "inactive")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from websites where uuid=$1", uuidReadOnly)
		db.Exec("delete from websites where uuid=$1", uuidInactive)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		userUUID    string
		group       string
		expect      model.WebsiteGroup
		expectError error
	}{
		{
			name:     "find active web of existing group and user",
			userUUID: userUUID,
			group:    title,
			expect: model.WebsiteGroup{
				{
					UserUUID:    userUUID,
					WebsiteUUID: uuid,
					GroupName:   title,
					AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Website: model.Website{
						UUID:       uuid,
						URL:        "http://example.com/" + title,
						Title:      title,
						UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
						Conf:       &config.WebsiteConfig{},
					},
				},
			},
			expectError: nil,
		},
		{
			name:     "find read-only web of existing group and user",
			userUUID: userUUID,
			group:    title + "-readonly",
			expect: model.WebsiteGroup{
				{
					UserUUID:    userUUID,
					WebsiteUUID: uuidReadOnly,
					GroupName:   title + "-readonly",
					AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Website: model.Website{
						UUID:       uuidReadOnly,
						URL:        "http://example.com/" + title + "-readonly",
						Title:      title + "-readonly",
						UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
						Conf:       &config.WebsiteConfig{},
					},
				},
			},
			expectError: nil,
		},
		{
			name:        "find inactive web of existing group and user",
			userUUID:    userUUID,
			group:       title + "-inactive",
			expect:      model.WebsiteGroup{},
			expectError: nil,
		},
		{
			name:        "find web of not existing group",
			userUUID:    userUUID,
			group:       "not exist",
			expect:      model.WebsiteGroup{},
			expectError: nil,
		},
		{
			name:        "find web of not existing user",
			userUUID:    "not exist",
			group:       title,
			expect:      model.WebsiteGroup{},
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := r.FindUserWebsitesByGroup(context.Background(), test.userUUID, test.group)
			assert.ErrorIs(t, err, test.expectError)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestSqlcRepo_FindUserWebsite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("postgres", connString)
	if err != nil {
		t.Fatalf("open database fail: %v", err)
	}

	r := NewRepo(db, &config.WebsiteConfig{})

	uuid := "find-user-website-uuid"
	userUUID := "find-user-website-user-uuid"
	title := "find user website"
	populateData(db, uuid, title, userUUID, "active")
	uuidReadOnly := "find-user-website-readonly-uuid"
	populateData(db, uuidReadOnly, title+"-readonly", userUUID, "read_only")
	uuidInactive := "find-user-website-inactive-uuid"
	populateData(db, uuidInactive, title+"-inactive", userUUID, "inactive")
	t.Cleanup(func() {
		db.Exec("delete from websites where uuid=$1", uuid)
		db.Exec("delete from websites where uuid=$1", uuidReadOnly)
		db.Exec("delete from websites where uuid=$1", uuidInactive)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	tests := []struct {
		name        string
		userUUID    string
		webUUID     string
		expect      *model.UserWebsite
		expectError error
	}{
		{
			name:     "find active web of existing group and user",
			userUUID: userUUID,
			webUUID:  uuid,
			expect: &model.UserWebsite{
				UserUUID:    userUUID,
				WebsiteUUID: uuid,
				GroupName:   title,
				AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Website: model.Website{
					UUID:       uuid,
					URL:        "http://example.com/" + title,
					Title:      title,
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectError: nil,
		},
		{
			name:     "find read-only web of existing group and user",
			userUUID: userUUID,
			webUUID:  uuidReadOnly,
			expect: &model.UserWebsite{
				UserUUID:    userUUID,
				WebsiteUUID: uuidReadOnly,
				GroupName:   title + "-readonly",
				AccessTime:  time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Website: model.Website{
					UUID:       uuidReadOnly,
					URL:        "http://example.com/" + title + "-readonly",
					Title:      title + "-readonly",
					UpdateTime: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectError: nil,
		},
		{
			name:        "find inactive web of existing group and user",
			userUUID:    userUUID,
			webUUID:     uuidInactive,
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
		{
			name:        "find web of not existing web uuid",
			userUUID:    userUUID,
			webUUID:     "not exist",
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
		{
			name:        "find web of not existing user",
			userUUID:    "not exist",
			webUUID:     uuid,
			expect:      nil,
			expectError: sql.ErrNoRows,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := r.FindUserWebsite(context.Background(), test.userUUID, test.webUUID)
			assert.ErrorIs(t, err, test.expectError)
			assert.Equal(t, test.expect, result)
		})
	}
}
