package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"nyamimo-go/client"
)

type LazySectionConfig struct {
	ID    string
	Title string
}

type SectionViewData struct {
	ID          string
	Title       string
	Variant     string
	SeeAllHref  string
	AnimeList   []client.AnimeItem
}

type HomePageData struct {
	Title            string
	CurrentPage      string
	User             *User
	HeroAnime        []client.AnimeItem
	OngoingSection   SectionViewData
	CompletedSection SectionViewData
	LazySections     []LazySectionConfig
}

type DetailPageData struct {
	Title              string
	CurrentPage        string
	User               *User
	Slug               string
	Detail             client.AnimeDetailData
	BatchDownloads     []client.DownloadFormat
	ActiveEpisodeNum   string
	ActiveEpisodeTitle string
	ActiveVideoURL     string
	ActiveRawIframe    template.HTML
	RawIframe          template.HTML
	VideoURL           string
	Videos             []client.PlayerOption
	GroupedVideos      map[string][]client.PlayerOption
}

type PopularPageData struct {
	Title        string
	CurrentPage  string
	User         *User
	CurrentOrder string
	Orders       []client.OrderOption
	AnimeList    []client.AnimeItem
	SearchQuery  string
}

type GenresPageData struct {
	Title          string
	CurrentPage    string
	User           *User
	CurrentGenreID string
	SelectedGenre  *client.Genre
	Genres         []client.Genre
	AnimeList      []client.AnimeItem
}

type TypePageData struct {
	Title       string
	CurrentPage string
	User        *User
	CurrentType string
	TypeName    string
	AnimeList   []client.AnimeItem
}

type SchedulePageData struct {
	Title       string
	CurrentPage string
	User        *User
	CurrentDay  string
	AnimeList   []client.AnimeItem
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"` // "admin" or "user"
}

type ProfilePageData struct {
	Title       string
	CurrentPage string
	User        *User
}

type AuthPageData struct {
	Title        string
	CurrentPage  string
	User         *User
	ErrorMessage string
}

type ModalPlayerData struct {
	EpisodeNum    string
	Title         string
	VideoURL      string
	RawIframe     template.HTML
	Videos        []client.PlayerOption
	GroupedVideos map[string][]client.PlayerOption
	Downloads     []client.DownloadFormat
}

var api *client.APIClient
var customHeroCarousel []client.AnimeItem

var defaultHDHeroAnime = []client.AnimeItem{
	{
		Title:    "Mushoku Tensei: Isekai Ittara Honki Dasu Season 3",
		Slug:     "mushoku-tensei-isekai-ittara-honki-dasu-season-3",
		Img:      "https://wallpapercat.com/w/full/8/9/a/25114-1920x1080-desktop-full-hd-mushoku-tensei-jobless-reincarnation-wallpaper-image.jpg",
		Episode:  "Episode 12",
		Score:    "8.7",
		Type:     "TV Series",
		Released: "2024",
	},
	{
		Title:    "One Piece",
		Slug:     "one-piece",
		Img:      "https://wallpapercat.com/w/full/4/1/0/33422-3840x2160-desktop-4k-one-piece-background.jpg",
		Episode:  "Episode 1178",
		Score:    "8.9",
		Type:     "TV Series",
		Released: "1999",
	},
	{
		Title:    "K-On!",
		Slug:     "k-on",
		Img:      "https://wallpapercat.com/w/full/7/d/a/816753-1920x1080-desktop-full-hd-k-on-wallpaper.jpg",
		Episode:  "Episode 13",
		Score:    "8.5",
		Type:     "TV Series",
		Released: "2009",
	},
	{
		Title:    "One Piece: Wano Arc",
		Slug:     "one-piece",
		Img:      "https://wallpapercat.com/w/full/3/3/6/126937-3840x2160-desktop-4k-one-piece-background-image.jpg",
		Episode:  "Episode 1071",
		Score:    "9.1",
		Type:     "TV Series",
		Released: "2023",
	},
	{
		Title:    "K-On! Live Concert",
		Slug:     "k-on",
		Img:      "https://wallpapercat.com/w/full/1/b/b/816777-3840x2160-desktop-4k-k-on-background-photo.jpg",
		Episode:  "Special",
		Score:    "8.8",
		Type:     "TV Series",
		Released: "2011",
	},
}

var (
	usersDb = map[string]User{
		"admin": {
			Username: "admin",
			Password: "admin123",
			Name:     "Administrator Nyamimo",
			Role:     "admin",
		},
		"user": {
			Username: "user",
			Password: "user123",
			Name:     "Member Nyamimo",
			Role:     "user",
		},
	}
	usersDbLock sync.RWMutex
)

func getLoggedInUser(r *http.Request) *User {
	cookie, err := r.Cookie("user_session")
	if err != nil || cookie.Value == "" {
		return nil
	}
	usersDbLock.RLock()
	defer usersDbLock.RUnlock()
	if u, ok := usersDb[cookie.Value]; ok {
		return &u
	}
	return nil
}

func main() {
	// Initialize API client with 10-minute cache TTL
	api = client.NewAPIClient(10 * time.Minute)

	mux := http.NewServeMux()

	// Page Routes
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/anime/", handleAnimeDetail)
	mux.HandleFunc("/popular", handlePopular)
	mux.HandleFunc("/search", handleSearchPage)
	mux.HandleFunc("/genres", handleGenres)
	mux.HandleFunc("/genres/", handleGenreDetail)
	mux.HandleFunc("/type", handleType)
	mux.HandleFunc("/type/", handleTypeDetail)
	mux.HandleFunc("/schedule", handleSchedule)
	mux.HandleFunc("/profile", handleProfile)
	mux.HandleFunc("/admin/carousel", handleAdminCarousel)

	// Auth Routes
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/api/login", handleLoginAPI)
	mux.HandleFunc("/api/google-login", handleGoogleLoginAPI)
	mux.HandleFunc("/register", handleRegister)
	mux.HandleFunc("/api/register", handleRegisterAPI)
	mux.HandleFunc("/logout", handleLogout)

	// Admin API Endpoints
	mux.HandleFunc("/api/admin/carousel/add", handleAdminCarouselAdd)
	mux.HandleFunc("/api/admin/carousel/delete", handleAdminCarouselDelete)
	mux.HandleFunc("/api/admin/carousel/reset", handleAdminCarouselReset)
	mux.HandleFunc("/api/admin/wallpaper-search", handleWallpaperSearch)

	// HTMX Partial API Endpoints
	mux.HandleFunc("/api/section/genre", handleSectionGenre)
	mux.HandleFunc("/api/search-suggest", handleSearchSuggest)
	mux.HandleFunc("/api/notifications", handleNotifications)
	mux.HandleFunc("/api/episode-modal", handleEpisodeModal)
	mux.HandleFunc("/api/episode-inline", handleEpisodeInline)
	// Static Files (Logo, Assets)
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/logo.png")
	})

	port := "3000"
	fmt.Printf("🚀 Server Go + HTMX running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// Template Helper FuncMap
var funcMap = template.FuncMap{
	"mod": func(i, j int) int {
		return i % j
	},
	"add": func(i, j int) int {
		return i + j
	},
	"cleanHDImg": client.GetCleanHDImage,
	"firstChar": func(s string) string {
		if len(s) == 0 {
			return "U"
		}
		return strings.ToUpper(string([]rune(s)[0]))
	},
	"getTotalAnimeCount": func() string {
		count := api.GetTotalAnimeCount()
		return fmt.Sprintf("%d+", count)
	},
}

// Render Helper with layout
func renderPage(w http.ResponseWriter, pageTemplate string, data interface{}) {
	tmpl, err := template.New("layout").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/navbar.html",
		"templates/footer.html",
		"templates/partials/section.html",
		"templates/"+pageTemplate,
	)
	if err != nil {
		http.Error(w, "Template render error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Printf("Execute error: %v", err)
	}
}

// Render Partial Helper
func renderPartial(w http.ResponseWriter, partialTemplate string, templateName string, data interface{}) {
	tmpl, err := template.New(templateName).Funcs(funcMap).ParseFiles("templates/partials/" + partialTemplate)
	if err != nil {
		http.Error(w, "Partial error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, templateName, data)
}

// Handlers
func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var newAnimeResp client.AnimeListResponse
	_ = api.GetJSON("/new-anime", &newAnimeResp)

	var ongoingResp client.AnimeListResponse
	_ = api.GetJSON("/ongoing-anime", &ongoingResp)

	var completedResp client.AnimeListResponse
	_ = api.GetJSON("/completed-anime", &completedResp)

	var genresResp client.GenreListResponse
	_ = api.GetJSON("/genres", &genresResp)

	heroAnime := customHeroCarousel
	if len(heroAnime) == 0 {
		heroAnime = defaultHDHeroAnime
	}

	for i := range ongoingResp.Data {
		ongoingResp.Data[i].Slug = client.GetAnimeSlug(ongoingResp.Data[i])
		ongoingResp.Data[i].Score = client.FormatScore(ongoingResp.Data[i].Score)
	}

	for i := range completedResp.Data {
		completedResp.Data[i].Slug = client.GetAnimeSlug(completedResp.Data[i])
		completedResp.Data[i].Score = client.FormatScore(completedResp.Data[i].Score)
	}

	var lazySections []LazySectionConfig
	for _, g := range genresResp.Data {
		lazySections = append(lazySections, LazySectionConfig{
			ID:    g.ID,
			Title: g.Title,
		})
	}

	data := HomePageData{
		Title:       "Nonton Anime Subtitle Indonesia Gratis HD",
		CurrentPage: "home",
		User:        getLoggedInUser(r),
		HeroAnime:   heroAnime,
		OngoingSection: SectionViewData{
			ID:         "ongoing",
			Title:      "On Going Anime",
			Variant:    "ongoing",
			SeeAllHref: "/popular?order=latest-update",
			AnimeList:  ongoingResp.Data,
		},
		CompletedSection: SectionViewData{
			ID:         "completed",
			Title:      "Completed Anime",
			Variant:    "completed",
			SeeAllHref: "/popular?order=popular",
			AnimeList:  completedResp.Data,
		},
		LazySections: lazySections,
	}

	renderPage(w, "index.html", data)
}

func handleAnimeDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/anime/")
	slug = strings.Trim(slug, "/")
	if slug == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var detail client.AnimeDetailData
	err := api.GetJSON("/detail-anime/"+slug, &detail)
	if err != nil || detail.Title == "" {
		var detailWrapper client.AnimeDetailResponse
		_ = api.GetJSON("/detail-anime/"+slug, &detailWrapper)
		detail = detailWrapper.Data
	}

	if detail.Title == "" {
		http.Error(w, "Anime not found", http.StatusNotFound)
		return
	}

	if detail.Synopsis == "" && len(detail.Descriptions) > 0 {
		detail.Synopsis = strings.Join(detail.Descriptions, "\n\n")
	}

	if detail.Rating != "" {
		detail.Score = detail.Rating
	} else {
		detail.Score = client.FormatScore(detail.Score)
	}

	for i := range detail.Genres {
		if detail.Genres[i].Title == "" && detail.Genres[i].Tag != "" {
			detail.Genres[i].Title = detail.Genres[i].Tag
		}
		if detail.Genres[i].ID == "" && detail.Genres[i].Link != "" {
			parts := strings.Split(strings.Trim(detail.Genres[i].Link, "/"), "/")
			if len(parts) > 0 {
				detail.Genres[i].ID = parts[len(parts)-1]
			}
		}
	}

	for i := range detail.Episodes {
		detail.Episodes[i].Number = client.FormatEpisodeNum(detail.Episodes[i].Episode)
	}

	for i := range detail.Recommendations {
		detail.Recommendations[i].Slug = client.GetAnimeSlug(detail.Recommendations[i])
		detail.Recommendations[i].Score = client.FormatScore(detail.Recommendations[i].Score)
	}

	var downloads []client.DownloadFormat
	var dlResp struct {
		Downloads []client.DownloadFormat `json:"downloads"`
	}
	_ = api.GetJSON("/download-anime/"+slug, &dlResp)
	downloads = dlResp.Downloads

	// Default active episode video loading for inline player
	activeEpIdx := 0
	epParam := r.URL.Query().Get("ep")
	if epParam != "" {
		for idx, epItem := range detail.Episodes {
			if epItem.Number == epParam || epItem.Episode == epParam {
				activeEpIdx = idx
				break
			}
		}
	} else if len(detail.Episodes) > 0 {
		// Default to Episode 1 (Episode Awal) instead of latest episode
		foundFirstEp := false
		for idx, epItem := range detail.Episodes {
			if epItem.Number == "1" || epItem.Episode == "1" || epItem.Number == "01" {
				activeEpIdx = idx
				foundFirstEp = true
				break
			}
		}
		if !foundFirstEp {
			// Fallback to the last element if list is sorted descending
			activeEpIdx = len(detail.Episodes) - 1
		}
	}

	var activeVideoURL string
	var activeRawIframe template.HTML
	var activeVideos []client.PlayerOption
	var activeGroupedVideos map[string][]client.PlayerOption
	activeEpNum := "1"
	activeEpTitle := ""

	detail.Title = client.CleanAnimeTitle(detail.Title)

	if len(detail.Episodes) > 0 && activeEpIdx < len(detail.Episodes) {
		targetEp := detail.Episodes[activeEpIdx]
		activeEpNum = targetEp.Number
		
		subTitle := client.CleanAnimeTitle(targetEp.Title)
		if strings.Contains(subTitle, detail.Title) {
			subTitle = strings.ReplaceAll(subTitle, detail.Title, "")
		}
		if idx := strings.Index(strings.ToLower(subTitle), "episode"); idx != -1 {
			subTitle = subTitle[:idx]
		}
		subTitle = strings.Trim(strings.TrimSpace(subTitle), " :-—–")
		activeEpTitle = subTitle

		var epsDetail client.EpisodeDetailResponse
		_ = api.GetJSON(targetEp.DetailEps, &epsDetail)

		if len(epsDetail.Videos) > 0 {
			activeVideos = epsDetail.Videos
			var vidResp struct {
				URL      string `json:"url"`
				Response string `json:"response"`
			}
			_ = api.GetJSON(epsDetail.Videos[0].Video, &vidResp)
			activeVideoURL = vidResp.URL
			activeRawIframe = template.HTML(vidResp.Response)
		} else if epsDetail.VideoURL != "" && epsDetail.VideoURL != "belum tersedia (segera)" {
			activeVideoURL = epsDetail.VideoURL
		}

		activeGroupedVideos = make(map[string][]client.PlayerOption)
		for _, v := range epsDetail.Videos {
			fields := strings.Fields(v.Title)
			provider := "Server Video"
			if len(fields) > 0 {
				provider = fields[0]
			}
			activeGroupedVideos[provider] = append(activeGroupedVideos[provider], v)
		}
	}

	data := DetailPageData{
		Title:              detail.Title,
		CurrentPage:        "detail",
		User:               getLoggedInUser(r),
		Slug:               slug,
		Detail:             detail,
		BatchDownloads:     downloads,
		ActiveEpisodeNum:   activeEpNum,
		ActiveEpisodeTitle: activeEpTitle,
		ActiveVideoURL:     activeVideoURL,
		ActiveRawIframe:    activeRawIframe,
		RawIframe:          activeRawIframe,
		VideoURL:           activeVideoURL,
		Videos:             activeVideos,
		GroupedVideos:      activeGroupedVideos,
	}

	renderPage(w, "anime_detail.html", data)
}

func handlePopular(w http.ResponseWriter, r *http.Request) {
	order := r.URL.Query().Get("order")
	if order == "" {
		order = "popular"
	}

	var ordersResp client.OrderListResponse
	_ = api.GetJSON("/available-orders", &ordersResp)

	if len(ordersResp.Data) == 0 {
		ordersResp.Data = []client.OrderOption{
			{Title: "Populer", Order: "popular"},
			{Title: "Rating", Order: "rating"},
			{Title: "Terbaru", Order: "latest-update"},
		}
	}

	var listResp client.AnimeListResponse
	_ = api.GetJSON("/order-anime/"+order, &listResp)

	for i := range listResp.Data {
		listResp.Data[i].Slug = client.GetAnimeSlug(listResp.Data[i])
		listResp.Data[i].Score = client.FormatScore(listResp.Data[i].Score)
	}

	data := PopularPageData{
		Title:        "Browse & Popular Anime",
		CurrentPage:  "popular",
		CurrentOrder: order,
		Orders:       ordersResp.Data,
		AnimeList:    listResp.Data,
	}

	renderPage(w, "popular.html", data)
}

func handleGenres(w http.ResponseWriter, r *http.Request) {
	var genresResp client.GenreListResponse
	_ = api.GetJSON("/genres", &genresResp)

	data := GenresPageData{
		Title:       "Daftar Genre Anime",
		CurrentPage: "genres",
		Genres:      genresResp.Data,
	}

	renderPage(w, "genres.html", data)
}

func handleGenreDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/genres/")

	var genresResp client.GenreListResponse
	_ = api.GetJSON("/genres", &genresResp)

	var selected *client.Genre
	for _, g := range genresResp.Data {
		if g.ID == id {
			selected = &g
			break
		}
	}

	var listResp client.AnimeListResponse
	_ = api.GetJSON("/genre-anime/"+id, &listResp)

	for i := range listResp.Data {
		listResp.Data[i].Slug = client.GetAnimeSlug(listResp.Data[i])
		listResp.Data[i].Score = client.FormatScore(listResp.Data[i].Score)
	}

	data := GenresPageData{
		Title:          "Genre " + id,
		CurrentPage:    "genres",
		CurrentGenreID: id,
		SelectedGenre:  selected,
		Genres:         genresResp.Data,
		AnimeList:      listResp.Data,
	}

	renderPage(w, "genres.html", data)
}

func handleType(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/type/tv", http.StatusSeeOther)
}

func handleTypeDetail(w http.ResponseWriter, r *http.Request) {
	t := strings.TrimPrefix(r.URL.Path, "/type/")
	if t == "" {
		t = "tv"
	}

	var listResp client.AnimeListResponse
	_ = api.GetJSON("/type-anime/"+t, &listResp)

	for i := range listResp.Data {
		listResp.Data[i].Slug = client.GetAnimeSlug(listResp.Data[i])
		listResp.Data[i].Score = client.FormatScore(listResp.Data[i].Score)
	}

	data := TypePageData{
		Title:       "Format Tipe: " + strings.ToUpper(t),
		CurrentPage: "type",
		CurrentType: t,
		TypeName:    strings.ToUpper(t),
		AnimeList:   listResp.Data,
	}

	renderPage(w, "type.html", data)
}

func handleSchedule(w http.ResponseWriter, r *http.Request) {
	dayMap := map[string]string{
		"senin":   "monday",
		"selasa":  "tuesday",
		"rabu":    "wednesday",
		"kamis":   "thursday",
		"jumat":   "friday",
		"sabtu":   "saturday",
		"minggu":  "sunday",
	}

	weekdayIndo := map[time.Weekday]string{
		time.Monday:    "senin",
		time.Tuesday:   "selasa",
		time.Wednesday: "rabu",
		time.Thursday:  "kamis",
		time.Friday:    "jumat",
		time.Saturday:  "sabtu",
		time.Sunday:    "minggu",
	}

	day := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("day")))
	if day == "" {
		day = weekdayIndo[time.Now().Weekday()]
		if day == "" {
			day = "senin"
		}
	}

	dayValue := dayMap[day]
	if dayValue == "" {
		dayValue = "monday"
		day = "senin"
	}

	var schedResp client.ReleaseScheduleResponse
	err := api.GetJSON("/release-schedule?day="+dayValue, &schedResp)
	if err != nil {
		log.Printf("Schedule API error: %v", err)
	}

	list := schedResp.Data
	for i := range list {
		list[i].Slug = client.GetAnimeSlug(list[i])
		list[i].Score = client.FormatScore(list[i].Score)
	}

	data := SchedulePageData{
		Title:       "Jadwal Rilis Anime",
		CurrentPage: "schedule",
		CurrentDay:  day,
		AnimeList:   list,
	}

	renderPage(w, "schedule.html", data)
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	user := getLoggedInUser(r)
	data := ProfilePageData{
		Title:       "My List & Profil Saya",
		CurrentPage: "profile",
		User:        user,
	}
	renderPage(w, "profile.html", data)
}

// HTMX Intersect Handler for Genre Carousels
func handleSectionGenre(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	title := r.URL.Query().Get("title")

	var listResp client.AnimeListResponse
	_ = api.GetJSON("/genre-anime/"+id, &listResp)

	for i := range listResp.Data {
		listResp.Data[i].Slug = client.GetAnimeSlug(listResp.Data[i])
		listResp.Data[i].Score = client.FormatScore(listResp.Data[i].Score)
	}

	data := SectionViewData{
		ID:         id,
		Title:      title,
		SeeAllHref: "/genres/" + id,
		AnimeList:  listResp.Data,
	}

	renderPartial(w, "section.html", "section", data)
}

// Full Search Page Handler
func handleSearchPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var results []client.AnimeItem
	if strings.TrimSpace(q) != "" {
		var searchResp client.AnimeListResponse
		_ = api.GetJSON("/search-anime?search="+client.FormatSearchQuery(q), &searchResp)
		results = searchResp.Data
		for i := range results {
			results[i].Slug = client.GetAnimeSlug(results[i])
			results[i].Score = client.FormatScore(results[i].Score)
			results[i].SeasonBadge = client.ExtractSeasonBadge(results[i].Title)
		}
	}

	data := PopularPageData{
		Title:        "Hasil Pencarian: " + q,
		CurrentPage:  "search",
		CurrentOrder: "",
		Orders:       nil,
		AnimeList:    results,
		SearchQuery:  q,
	}

	renderPage(w, "popular.html", data)
}

// HTMX Search Suggest Handler
func handleSearchSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	var results []client.AnimeItem
	var headerTitle string

	if q == "" {
		headerTitle = "Anime yang sedang tren"
		var popularResp client.AnimeListResponse
		_ = api.GetJSON("/order-anime/popular", &popularResp)
		results = popularResp.Data
		if len(results) == 0 {
			var ongoingResp client.AnimeListResponse
			_ = api.GetJSON("/ongoing-anime", &ongoingResp)
			results = ongoingResp.Data
		}
		if len(results) > 8 {
			results = results[:8]
		}
	} else {
		headerTitle = "Hasil Pencarian Anime"
		var searchResp client.AnimeListResponse
		_ = api.GetJSON("/search-anime?search="+client.FormatSearchQuery(q), &searchResp)
		results = searchResp.Data
		if len(results) > 8 {
			results = results[:8]
		}
	}

	for i := range results {
		results[i].Slug = client.GetAnimeSlug(results[i])
		results[i].Score = client.FormatScore(results[i].Score)
		results[i].SeasonBadge = client.ExtractSeasonBadge(results[i].Title)
	}

	data := struct {
		TitleHeader   string
		IsSearchQuery bool
		Results       []client.AnimeItem
	}{
		TitleHeader:   headerTitle,
		IsSearchQuery: q != "",
		Results:       results,
	}

	renderPartial(w, "search_results.html", "search_results", data)
}

// HTMX Notifications Handler
func handleNotifications(w http.ResponseWriter, r *http.Request) {
	var listResp client.AnimeListResponse
	_ = api.GetJSON("/order-anime/latest-update", &listResp)

	results := listResp.Data
	if len(results) > 8 {
		results = results[:8]
	}

	for i := range results {
		results[i].Slug = client.GetAnimeSlug(results[i])
		results[i].Score = client.FormatScore(results[i].Score)
	}

	data := struct {
		Results []client.AnimeItem
	}{
		Results: results,
	}

	renderPartial(w, "search_results.html", "search_results", data)
}

// HTMX Episode Modal Handler
func handleEpisodeModal(w http.ResponseWriter, r *http.Request) {
	detailEps := r.URL.Query().Get("detail_eps")
	title := r.URL.Query().Get("title")
	ep := r.URL.Query().Get("ep")

	var epsDetail client.EpisodeDetailResponse
	_ = api.GetJSON(detailEps, &epsDetail)

	var firstVideoURL string
	var firstIframe template.HTML
	if len(epsDetail.Videos) > 0 {
		var vidResp struct {
			URL      string `json:"url"`
			Response string `json:"response"`
		}
		_ = api.GetJSON(epsDetail.Videos[0].Video, &vidResp)
		firstVideoURL = vidResp.URL
		firstIframe = template.HTML(vidResp.Response)
	} else if epsDetail.VideoURL != "" && epsDetail.VideoURL != "belum tersedia (segera)" {
		firstVideoURL = epsDetail.VideoURL
	}

	// Group videos by server provider name
	grouped := make(map[string][]client.PlayerOption)
	for _, v := range epsDetail.Videos {
		fields := strings.Fields(v.Title)
		provider := "Server Video"
		if len(fields) > 0 {
			provider = fields[0]
		}
		grouped[provider] = append(grouped[provider], v)
	}

	data := ModalPlayerData{
		EpisodeNum:    ep,
		Title:         title,
		VideoURL:      firstVideoURL,
		RawIframe:     firstIframe,
		Videos:        epsDetail.Videos,
		GroupedVideos: grouped,
		Downloads:     epsDetail.Downloads,
	}

	renderPartial(w, "modal_player.html", "modal_player", data)
}

// HTMX Episode Inline Handler for Detail Page Player Swap
func handleEpisodeInline(w http.ResponseWriter, r *http.Request) {
	detailEps := r.URL.Query().Get("detail_eps")
	title := r.URL.Query().Get("title")
	ep := r.URL.Query().Get("ep")

	var epsDetail client.EpisodeDetailResponse
	_ = api.GetJSON(detailEps, &epsDetail)

	var firstVideoURL string
	var firstIframe template.HTML
	if len(epsDetail.Videos) > 0 {
		var vidResp struct {
			URL      string `json:"url"`
			Response string `json:"response"`
		}
		_ = api.GetJSON(epsDetail.Videos[0].Video, &vidResp)
		firstVideoURL = vidResp.URL
		firstIframe = template.HTML(vidResp.Response)
	} else if epsDetail.VideoURL != "" && epsDetail.VideoURL != "belum tersedia (segera)" {
		firstVideoURL = epsDetail.VideoURL
	}

	grouped := make(map[string][]client.PlayerOption)
	for _, v := range epsDetail.Videos {
		fields := strings.Fields(v.Title)
		provider := "Server Video"
		if len(fields) > 0 {
			provider = fields[0]
		}
		grouped[provider] = append(grouped[provider], v)
	}

	data := ModalPlayerData{
		EpisodeNum:    ep,
		Title:         title,
		VideoURL:      firstVideoURL,
		RawIframe:     firstIframe,
		Videos:        epsDetail.Videos,
		GroupedVideos: grouped,
		Downloads:     epsDetail.Downloads,
	}

	renderPartial(w, "anime_detail.html", "inline_player_area", data)
}

// HTMX Video URL Switcher Handler
func handleVideoURL(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	var vidResp struct {
		URL      string `json:"url"`
		Response string `json:"response"`
	}
	_ = api.GetJSON(path, &vidResp)

	data := struct {
		VideoURL  string
		RawIframe template.HTML
	}{
		VideoURL:  vidResp.URL,
		RawIframe: template.HTML(vidResp.Response),
	}

	renderPartial(w, "modal_player.html", "iframe_player", data)
}

// Admin Carousel Management Handlers
func handleAdminCarousel(w http.ResponseWriter, r *http.Request) {
	currentUser := getLoggedInUser(r)
	if currentUser == nil || currentUser.Role != "admin" {
		data := AuthPageData{
			Title:        "Akses Ditolak - Role Admin Diperlukan",
			CurrentPage:  "login",
			User:         currentUser,
			ErrorMessage: "Akses Ditolak: Kamu harus masuk sebagai Admin (username: admin / pass: admin123) untuk mengelola Carousel.",
		}
		renderPage(w, "login.html", data)
		return
	}

	heroAnime := customHeroCarousel
	if len(heroAnime) == 0 {
		heroAnime = defaultHDHeroAnime
	}

	data := HomePageData{
		Title:       "Pengaturan Hero Carousel Banner - Admin",
		CurrentPage: "admin",
		User:        currentUser,
		HeroAnime:   heroAnime,
	}

	renderPage(w, "admin_carousel.html", data)
}

func handleAdminCarouselAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/carousel", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	title := r.FormValue("title")
	slug := r.FormValue("slug")
	img := r.FormValue("img")
	episode := r.FormValue("episode")
	score := r.FormValue("score")
	animeType := r.FormValue("type")

	newItem := client.AnimeItem{
		Title:   title,
		Slug:    slug,
		Img:     client.GetCleanHDImage(img),
		Episode: episode,
		Score:   score,
		Type:    animeType,
	}

	if len(customHeroCarousel) == 0 {
		var newAnimeResp client.AnimeListResponse
		_ = api.GetJSON("/new-anime", &newAnimeResp)
		items := newAnimeResp.Data
		if len(items) > 8 {
			items = items[:8]
		}
		for i := range items {
			items[i].Slug = client.GetAnimeSlug(items[i])
			items[i].Score = client.FormatScore(items[i].Score)
			items[i].Img = client.GetCleanHDImage(items[i].Img)
		}
		customHeroCarousel = items
	}

	customHeroCarousel = append([]client.AnimeItem{newItem}, customHeroCarousel...)
	http.Redirect(w, r, "/admin/carousel", http.StatusSeeOther)
}

func handleAdminCarouselDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/carousel", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	slug := r.FormValue("slug")

	if len(customHeroCarousel) == 0 {
		var newAnimeResp client.AnimeListResponse
		_ = api.GetJSON("/new-anime", &newAnimeResp)
		items := newAnimeResp.Data
		if len(items) > 8 {
			items = items[:8]
		}
		for i := range items {
			items[i].Slug = client.GetAnimeSlug(items[i])
			items[i].Score = client.FormatScore(items[i].Score)
			items[i].Img = client.GetCleanHDImage(items[i].Img)
		}
		customHeroCarousel = items
	}

	var filtered []client.AnimeItem
	for _, item := range customHeroCarousel {
		if item.Slug != slug {
			filtered = append(filtered, item)
		}
	}
	customHeroCarousel = filtered

	http.Redirect(w, r, "/admin/carousel", http.StatusSeeOther)
}

func handleAdminCarouselReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/carousel", http.StatusSeeOther)
		return
	}

	customHeroCarousel = nil
	http.Redirect(w, r, "/admin/carousel", http.StatusSeeOther)
}

// User & Admin Authentication Handlers
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if getLoggedInUser(r) != nil {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	renderPage(w, "login.html", AuthPageData{Title: "Masuk Akun", CurrentPage: "login"})
}

func handleLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	password := strings.TrimSpace(r.FormValue("password"))

	usersDbLock.RLock()
	user, exists := usersDb[username]
	usersDbLock.RUnlock()

	if !exists || user.Password != password {
		referer := r.Header.Get("Referer")
		if referer != "" && !strings.Contains(referer, "/login") {
			errMsg := url.QueryEscape("Username atau kata sandi salah. Gunakan admin/admin123 atau user/user123!")
			sep := "?"
			if strings.Contains(referer, "?") {
				sep = "&"
			}
			http.Redirect(w, r, referer+sep+"login_error="+errMsg, http.StatusSeeOther)
			return
		}
		data := AuthPageData{
			Title:        "Masuk Akun",
			CurrentPage:  "login",
			ErrorMessage: "Username atau kata sandi salah. Gunakan admin/admin123 atau user/user123!",
		}
		renderPage(w, "login.html", data)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    username,
		Path:     "/",
		HttpOnly: true,
	})

	referer := r.Header.Get("Referer")
	if referer == "" || strings.Contains(referer, "/login") {
		if user.Role == "admin" {
			referer = "/admin/carousel"
		} else {
			referer = "/"
		}
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func handleGoogleLoginAPI(w http.ResponseWriter, r *http.Request) {
	username := "google_user"
	usersDbLock.Lock()
	if _, exists := usersDb[username]; !exists {
		usersDb[username] = User{
			Username: username,
			Password: "google_account",
			Name:     "Pengguna Google",
			Role:     "user",
		}
	}
	usersDbLock.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    username,
		Path:     "/",
		HttpOnly: true,
	})

	referer := r.Header.Get("Referer")
	if referer == "" || strings.Contains(referer, "/login") {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "register.html", AuthPageData{Title: "Daftar Akun Member", CurrentPage: "register"})
}

func handleRegisterAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	password := strings.TrimSpace(r.FormValue("password"))

	if username == "" || password == "" {
		renderPage(w, "register.html", AuthPageData{Title: "Daftar Akun", CurrentPage: "register", ErrorMessage: "Mohon isi semua bidang formulir!"})
		return
	}

	usersDbLock.Lock()
	if _, exists := usersDb[username]; exists {
		usersDbLock.Unlock()
		renderPage(w, "register.html", AuthPageData{Title: "Daftar Akun", CurrentPage: "register", ErrorMessage: "Username sudah terdaftar! Gunakan username lain."})
		return
	}

	newUser := User{
		Username: username,
		Password: password,
		Name:     name,
		Role:     "user",
	}
	usersDb[username] = newUser
	usersDbLock.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    username,
		Path:     "/",
		HttpOnly: true,
	})

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

type WallpaperItem struct {
	URL        string `json:"url"`
	Resolution string `json:"resolution"`
	Title      string `json:"title"`
}

func fetchWallpaperCat(slug string) ([]WallpaperItem, error) {
	targetURL := "https://wallpapercat.com/" + slug
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyStr := string(bodyBytes)

	re := regexp.MustCompile(`/w/full/[a-zA-Z0-9/_.-]+\.(jpg|png|jpeg|webp)`)
	matches := re.FindAllString(bodyStr, -1)

	var items []WallpaperItem
	seen := make(map[string]bool)

	for _, m := range matches {
		parts := strings.Fields(m)
		if len(parts) > 0 {
			m = parts[0]
		}
		if seen[m] {
			continue
		}
		seen[m] = true

		fullURL := "https://wallpapercat.com" + m
		res := "Full HD 1080p"
		if strings.Contains(m, "3840x2160") || strings.Contains(m, "4k") || strings.Contains(m, "7680x4320") {
			res = "4K Ultra HD"
		} else if strings.Contains(m, "2560x1440") || strings.Contains(m, "2560x1600") {
			res = "2K QHD"
		}

		if strings.Contains(m, "mobile") {
			continue
		}

		items = append(items, WallpaperItem{
			URL:        fullURL,
			Resolution: res,
			Title:      slug,
		})

		if len(items) >= 8 {
			break
		}
	}

	return items, nil
}

func slugifyAnimeTitle(title string) string {
	t := strings.ToLower(title)
	if idx := strings.Index(t, ":"); idx != -1 {
		t = t[:idx]
	}
	t = strings.ReplaceAll(t, "!", "")
	t = strings.ReplaceAll(t, "?", "")
	t = strings.ReplaceAll(t, ",", "")
	t = strings.ReplaceAll(t, ".", "")
	t = strings.ReplaceAll(t, "'", "")
	t = strings.TrimSpace(t)

	words := strings.Fields(t)
	var cleanWords []string
	for _, w := range words {
		if w == "season" || w == "arc" || w == "movie" || w == "tv" || w == "series" || w == "sub" || w == "indo" {
			break
		}
		cleanWords = append(cleanWords, w)
	}
	if len(cleanWords) == 0 {
		return "anime"
	}
	return strings.Join(cleanWords, "-")
}

func handleWallpaperSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		q = r.URL.Query().Get("title")
	}
	q = strings.TrimSpace(q)

	var items []WallpaperItem
	if q != "" {
		slugBase := slugifyAnimeTitle(q)
		candidates := []string{
			slugBase + "-wallpapers",
			slugBase + "-wallpaper",
			slugBase,
			slugBase + "-backgrounds",
		}

		for _, cand := range candidates {
			found, err := fetchWallpaperCat(cand)
			if err == nil && len(found) > 0 {
				items = found
				break
			}
		}
	}

	if len(items) == 0 {
		items = []WallpaperItem{
			{URL: "https://wallpapercat.com/w/full/8/9/a/25114-1920x1080-desktop-full-hd-mushoku-tensei-jobless-reincarnation-wallpaper-image.jpg", Resolution: "Full HD 1080p", Title: "Mushoku Tensei Landscape"},
			{URL: "https://wallpapercat.com/w/full/4/1/0/33422-3840x2160-desktop-4k-one-piece-background.jpg", Resolution: "4K Ultra HD", Title: "One Piece 4K"},
			{URL: "https://wallpapercat.com/w/full/7/d/a/816753-1920x1080-desktop-full-hd-k-on-wallpaper.jpg", Resolution: "Full HD 1080p", Title: "K-On! Concert"},
			{URL: "https://wallpapercat.com/w/full/3/3/6/126937-3840x2160-desktop-4k-one-piece-background-image.jpg", Resolution: "4K Ultra HD", Title: "Luffy Gear 5 / Wano"},
			{URL: "https://wallpapercat.com/w/full/5/3/1/141742-3840x2160-desktop-4k-naruto-wallpaper-photo.jpg", Resolution: "4K Ultra HD", Title: "Naruto Shippuden 4K"},
			{URL: "https://wallpapercat.com/w/full/1/b/b/816777-3840x2160-desktop-4k-k-on-background-photo.jpg", Resolution: "4K Ultra HD", Title: "Anime Live Concert"},
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	titleLabel := q
	if titleLabel == "" {
		titleLabel = "Koleksi Rekomendasi"
	}
	fmt.Fprintf(w, `<div class="space-y-3 pt-3 border-t border-[#E2E2DC]">
		<div class="flex items-center justify-between">
			<span class="text-xs font-bold text-[#1A1A1E]">Pilih Wallpaper WallpaperCat HD ("%s"):</span>
			<span class="text-[11px] font-semibold text-[#55555B]">%d gambar HD</span>
		</div>
		<div class="grid grid-cols-2 gap-2.5 max-h-64 overflow-y-auto p-2 border border-[#E2E2DC] rounded-xl bg-[#F6F5F0]">`, template.HTMLEscapeString(titleLabel), len(items))

	for _, item := range items {
		fmt.Fprintf(w, `<div class="group relative rounded-lg overflow-hidden border border-[#E2E2DC] hover:border-[#FFCC00] cursor-pointer transition-all bg-black aspect-video shadow-sm" onclick="selectWallpaper('%s')">
			<img src="%s" alt="Wallpaper" class="w-full h-full object-cover group-hover:scale-105 transition-transform" />
			<span class="absolute top-1 left-1 px-1.5 py-0.5 rounded text-[9px] font-extrabold bg-[#FFCC00] text-[#17171B] shadow">
				%s
			</span>
			<div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
				<span class="px-2 py-1 rounded bg-[#FFCC00] text-[#17171B] text-[10px] font-extrabold shadow flex items-center gap-1">
					✓ Pilih Gambar Ini
				</span>
			</div>
		</div>`, template.HTMLEscapeString(item.URL), template.HTMLEscapeString(item.URL), template.HTMLEscapeString(item.Resolution))
	}

	fmt.Fprintf(w, `</div></div>`)
}


