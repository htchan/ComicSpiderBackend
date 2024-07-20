package sqlc

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
)

func BenchmarkPsqlRepo_CreateWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-create-website"

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Close()
	})

	updateTime := time.Now().UTC().Truncate(time.Second)

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		web := model.Website{
			UUID:       fmt.Sprintf("uuid-%v", n),
			URL:        "https://test.com",
			Title:      title,
			RawContent: "",
			UpdateTime: updateTime,
		}
		b.StartTimer()
		err := r.CreateWebsite(&web)
		b.StopTimer()
		if err != nil {
			b.Errorf("iteration-%d: create website return %v", n, err)
		}
	}
}

func BenchmarkPsqlRepo_UpdateWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-update-website"
	uuid := "benchmark-update-website-uuid"
	userUUID := "benchmark-update-website-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	web := model.Website{
		UUID:       uuid,
		URL:        "http://example.com/" + title,
		Title:      title,
		RawContent: "content new",
		UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		web.RawContent = fmt.Sprintf("content %v", n)
		b.StartTimer()
		err := r.UpdateWebsite(&web)
		b.StopTimer()
		if err != nil {
			b.Errorf("iteration-%d: update website return %v", n, err)
		}
	}
}

func BenchmarkPsqlRepo_DeleteWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-delete-website"
	uuid := "benchmark-delete-website-uuid"
	userUUID := "benchmark-delete-website-user-uuid"

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		populateData(db, uuid, title, userUUID)
		b.StartTimer()
		err := r.DeleteWebsite(&model.Website{UUID: uuid})
		b.StopTimer()
		if err != nil {
			b.Errorf("iteration-%d: delete website return %v", n, err)
		}
	}
}

func BenchmarkPsqlRepo_FindWebsites(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-find-websites"
	uuid := "benchmark-find-websites-uuid"
	userUUID := "benchmark-find-websites-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		b.StartTimer()
		webs, err := r.FindWebsites()
		b.StopTimer()
		if len(webs) == 0 || err != nil {
			b.Errorf("iteration-%d: find websites return %v; web count: %d", n, err, len(webs))
		}
	}
}

func BenchmarkPsqlRepo_FindWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-find-website"
	uuid := "benchmark-find-website-uuid"
	userUUID := "benchmark-find-website-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		b.StartTimer()
		web, err := r.FindWebsite(uuid)
		b.StopTimer()
		if web == nil || err != nil {
			b.Errorf("iteration-%d: find websites return %v; web: %v", n, err, web)
		}
	}
}

func BenchmarkPsqlRepo_CreateUserWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	uuid := "benchmark-create-user-website-uuid"
	title := "benchmark create user website"
	userUUID := "benchmark-create-user-website-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	accessTime := time.Now().UTC().Truncate(time.Second)

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		web := model.UserWebsite{
			WebsiteUUID: uuid,
			UserUUID:    fmt.Sprintf("benchmark-create-user-website-user-%d", n),
			GroupName:   "group",
			AccessTime:  accessTime,
		}
		b.StartTimer()
		err := r.CreateUserWebsite(&web)
		b.StopTimer()
		if err != nil {
			b.Errorf("iteration-%d: create user website return %v", n, err)
		}
	}
}

func BenchmarkPsqlRepo_UpdateUserWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-update-user-website"
	userUUID := "benchmark-update-user-website-user-uuid"
	uuid := "benchmark-update-user-website-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	web := model.UserWebsite{
		WebsiteUUID: uuid,
		UserUUID:    userUUID,
		GroupName:   "group",
		AccessTime:  time.Now().UTC().Truncate(time.Second),
	}

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		web.GroupName = fmt.Sprintf("group %v", n)
		b.StartTimer()
		err := r.UpdateUserWebsite(&web)
		b.StopTimer()
		if err != nil {
			b.Errorf("iteration-%d: update user website return %v", n, err)
		}
	}
}

func BenchmarkPsqlRepo_DeleteUserWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-delete-user-website"
	uuid := "benchmark-delete-user-website-uuid"
	userUUID := "benchmark-delete-user-website-user-uuid"

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		populateData(db, uuid, title, userUUID)
		b.StartTimer()
		err := r.DeleteUserWebsite(&model.UserWebsite{WebsiteUUID: uuid, UserUUID: userUUID})
		b.StopTimer()
		if err != nil {
			b.Errorf("iteration-%d: delete website return %v", n, err)
		}
	}
}

func BenchmarkPsqlRepo_FindUserWebsites(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-find-user-websites"
	uuid := "benchmark-find-user-websites-uuid"
	userUUID := "benchmark-find-user-websites-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		b.StartTimer()
		webs, err := r.FindUserWebsites(userUUID)
		b.StopTimer()
		if len(webs) == 0 || err != nil {
			b.Errorf("iteration-%d: find user websites return %v; web count: %d", n, err, len(webs))
		}
	}
}

func BenchmarkPsqlRepo_FindUserWebsitesByGroup(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-find-user-websites-by-group"
	uuid := "benchmark-find-user-websites-by-group-uuid"
	userUUID := "benchmark-find-user-websites-by-group-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		b.StartTimer()
		webs, err := r.FindUserWebsitesByGroup(userUUID, title)
		b.StopTimer()
		if len(webs) == 0 || err != nil {
			b.Errorf("iteration-%d: find user websites by group return %v; web count: %d", n, err, len(webs))
		}
	}
}

func BenchmarkPsqlRepo_FindUserWebsite(b *testing.B) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		b.Fatalf("open database fail: %v", err)
	}

	title := "benchmark-find-user-website"
	uuid := "benchmark-find-user-website-uuid"
	userUUID := "benchmark-find-user-website-user-uuid"
	populateData(db, uuid, title, userUUID)

	r := NewRepo(db, &config.WebsiteConfig{})
	b.Cleanup(func() {
		db.Exec("delete from websites where title=$1", title)
		db.Exec("delete from user_websites where website_uuid=$1", uuid)
		db.Close()
	})

	b.ResetTimer()
	b.StopTimer()
	for n := 0; n < b.N; n++ {
		b.StartTimer()
		web, err := r.FindUserWebsite(userUUID, uuid)
		b.StopTimer()
		if web == nil || err != nil {
			b.Errorf("iteration-%d: find websites return %v; web: %v", n, err, web)
		}
	}
}
