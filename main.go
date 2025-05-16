package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

const (
	BASE_URL      = "https://www.elitemadzone.org"
	POST_URI      = "/poruka.php"
	FLOOD_TIMEOUT = 31
	RUN_TIMEOUT   = 600
)

var ErrLockedTopic = errors.New("zaključana tema")

/**
* Za dati postId vraca Node koji sadrzi objavu
 */
func getPost(c *http.Client, postId string) (*html.Node, error) {
	url := BASE_URL + fmt.Sprintf("/p%s", postId)
	doc, err := getPage(c, url)
	if err != nil {
		return nil, err
	}
	post := htmlquery.FindOne(doc, fmt.Sprintf("//div[@id='%s']", "poruka_"+postId))
	return post, nil
}

/**
* Za dati url vraca Node objekat cele stranice
 */
func getPage(c *http.Client, url string) (*html.Node, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	//Konekcija je potrebna funkciji cookieHack koja se poziva u okviru CustomTransport-a
	ctx := context.WithValue(req.Context(), clientKey, c)
	req = req.WithContext(ctx)

	addHeaders(&req.Header)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

/**
* Proverava da li je post izbrisan ili ga je promenio neko ko nije autor posta
 */
func isPostOk(c *http.Client, postId string, allowedUser string) (bool, error) {
	post, err := getPost(c, postId)
	if err != nil {
		return false, err
	}
	if post == nil {
		return false, nil
	}
	//proveri da li je post menjao neko drugi
	for _, node := range htmlquery.Find(post, "./sub/b/text()") {
		sub := htmlquery.InnerText(node)
		r := regexp.MustCompile("^Ovu poruku je menjao (?P<user>.+) dana")
		m := r.FindStringSubmatch(sub)
		if m != nil {
			i := r.SubexpIndex("user")
			changedBy := m[i]
			if changedBy != allowedUser {
				return false, nil
			}
		}
	}
	return true, nil
}

/*
*
* Ucitava parametre koji identifikuju forum i temu, kao i es_token (verovatno csrf) i detektuje da li je tema zakljucana.
* Azurira postMetaT
forma:

subject:  Re: Srbija - raspada li se ili ne?
posticon: 1
message:  Nije potrebno more, svi avioni se obaraju raketama ili usmerenim snopom energije. Nama je potrebno Teslino oružje koje je još pre Drugog
svetskog rata nudio jugoslovenskoj vladi preko Ivana Meštrovića.
TopicID:  510739
BoardID:  2
es_token: MTc0Njc3MjAyMGhmZzRQS0Mwb29NcVdEeko1Vk1udW1ScVJSS3B5R3pY
Submit:   Pošalji odgovor

<input type="hidden" name="TopicID" value="510739" />
<input type="hidden" name="BoardID" value="2" />
<input type="hidden" name="es_token" value="MTc0Njc3MjAyMGhmZzRQS0Mwb29NcVdEeko1Vk1udW1ScVJSS3B5R3pY" />
*/

func getForumTopicData(c *http.Client, url string) (topicId string, boardId string, token string, locked bool, err error) {
	doc, err := getPage(c, url)
	if err != nil {
		return "", "", "", false, err
	}
	locked = false
	for _, node := range htmlquery.Find(doc, "//td[@class='msg1']/p[@class='tiny']/b") {
		str := htmlquery.InnerText(node)
		if strings.Contains(str, "Zaključana tema (lock)") {
			locked = true
			break
		}
	}

	if topicIdNode := htmlquery.FindOne(doc, "//input[@name='TopicID']/@value"); topicIdNode != nil {
		topicId = htmlquery.InnerText(topicIdNode)
	}

	if boardIdNode := htmlquery.FindOne(doc, "//input[@name='BoardID']/@value"); boardIdNode != nil {
		boardId = htmlquery.InnerText(boardIdNode)
	}
	if tokenNode := htmlquery.FindOne(doc, "//input[@name='es_token']/@value"); tokenNode != nil {
		token = htmlquery.InnerText(tokenNode)
	}
	return topicId, boardId, token, locked, nil
}

/**
* Salje objavu na forum. Nakon slanja neophodno je proveriti da li je objava prihvacena. Ova funkcija ne radi proveru, samo salje.
 */
func writePost(c *http.Client, post *postMetaT) error {
	time.Sleep(FLOOD_TIMEOUT * time.Second)
	pageUrl := BASE_URL + "/t" + post.TopicId
	topicId, boardId, token, locked, err := getForumTopicData(c, pageUrl)
	if err != nil {
		return err
	}
	if locked {
		return ErrLockedTopic
	}
	contentMessage, err := getPostContent(post)
	if err != nil {
		return err
	}
	form := url.Values{
		"subject":  {post.Title},
		"posticon": {"1"},
		"message":  {contentMessage},
		"TopicID":  {topicId},
		"BoardID":  {boardId},
		"es_token": {token},
		"Submit":   {"Pošalji odgovor"},
	}
	req, err := http.NewRequest("POST", BASE_URL+POST_URI, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	//Konekcija je potrebna funkciji cookieHack koja se poziva u okviru CustomTransport-a
	ctx := context.WithValue(req.Context(), clientKey, c)
	req = req.WithContext(ctx)

	addHeaders(&req.Header)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type postMetaT struct {
	Title    string `json:"naslov"`
	PostId   string `json:"postId"`
	TopicId  string `json:"temaId"`
	AuthorId string `json:"autor"`
	File     string `json:"fajl"`
}

type userT struct {
	Username string `json:"korisnik"`
	Password string `json:"lozinka"`
	Active   bool   `json:"aktivan"`
	posts    []*postMetaT
}

/**
* Ucitava sadrzaj posta iz lokalnog fajla, cija se putanja nalazi u okviru postMetaT::File
 */
func getPostContent(post *postMetaT) (string, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	path := baseDir + "/" + strings.TrimLeft(post.File, "/\\")
	content, err := os.ReadFile(path)
	if err != nil {
		log.Println("Greška " + err.Error())
		return "", err
	}
	return string(content), nil
}

/**
* Vraca postId poslednje objave na poslednjoj stranici teme za datog autora
 */
func getLastPostId(c *http.Client, topicId string, authorId string) (string, error) {
	url := BASE_URL + "/tema/poslednjastrana/" + topicId
	doc, err := getPage(c, url)
	if err != nil {
		return "", err
	}
	if doc == nil {
		return "", errors.New("prazan dokument za url " + url)
	}
	queryLastAuthor := "((//td[@id='posterinfo']/p[@class='tiny']/span[text()='" + authorId + "'])[last()]"
	node := htmlquery.FindOne(doc, "("+queryLastAuthor+")/../../..)//table[1]//a[1]/@href")
	if node == nil {
		return "", nil
	}
	postUri := htmlquery.InnerText(node) //uri je u obliku /p4101642
	postId := postUri[2:]
	return postId, nil
}

/**
* Nakon upisa neophodno je procitati idPost i upisati ga u postMetaT strukturu.
* Da bih znao da li je upis uspeo, pre upisa procitam id poslednjeg posta datog autora u datoj temi - ulazni parametar previousPostId.
* Nakon upisa ponovim citanje. Ako je id isti, to znaci da post nije upisan i vracam gresku.
 */
func updatePostId(c *http.Client, post *postMetaT, previousPostId string) error {
	newPostId, err := getLastPostId(c, post.TopicId, post.AuthorId)
	if err != nil {
		return err
	}
	if newPostId == "" || newPostId == previousPostId {
		return errors.New("post " + post.Title + " nije upisan")
	}

	post.PostId = newPostId
	return nil
}

/**
* Provera da li korisnik ima ogranicenje pristupa jer je moguce da nije banovan, ali ne moze da pise.
 */
func (user *userT) isLimitedUser(c *http.Client) (bool, error) {
	page, err := getPage(c, BASE_URL+"/korisnik/profil/"+url.PathEscape(user.Username))
	if err != nil {
		return false, err
	}
	html := htmlquery.OutputHTML(page, false)
	limited := strings.Contains(html, "<b>Ograničenje pristupa:</b>")
	return limited, nil
}

/**
* Proverava sve postove dodeljene korisniku i ako ne postoje ili su izmenjeni objavljuje ih ponovo
 */
func (user *userT) run() {
	if !user.Active {
		log.Printf("Neaktivni korisnik %s", user.Username)
		return
	}
	c := newClient()
	//prijava na sajt
	if err := login(c, LOGIN_URI, user.getLoginData()); err != nil {
		if errors.Is(err, ErrBanned) {
			user.Active = false
		}
		log.Printf("Greška prilikom prijave korisnika %s, %s", user.Username, err.Error())
		return
	}
	//provera da li korisnik ima ogranicenje
	limited, err := user.isLimitedUser(c)
	if err != nil {
		log.Printf("Greška prilikom provere da li korisnik %s ima ograničenja %s", user.Username, err.Error())
		return
	}
	if limited {
		user.Active = false
		log.Printf("Korisnik %s ima ograničenje", user.Username)
		return
	}
	//prolazak kroz sve postove pridruzene korisniku
	for _, post := range user.posts {
		postOk, err := isPostOk(c, post.PostId, post.AuthorId)
		if err != nil {
			log.Println("Greška " + err.Error())
			continue
		}
		if postOk {
			continue
		}
		log.Printf("Promenjen ili izbrisan post t%s, %s", post.PostId, post.Title)
		post.AuthorId = user.Username
		previousPostId, err := getLastPostId(c, post.TopicId, post.AuthorId)
		if err != nil {
			log.Println("Greška ", err.Error())
			continue
		}
		log.Println("Upisujem " + post.Title)
		if err = writePost(c, post); err != nil {
			log.Println("Greška ", err.Error())
			continue
		}
		err = updatePostId(c, post, previousPostId)
		if err != nil {
			log.Println("Greška " + err.Error())
			continue
		}
	}
}

/**
* Ucitava korisnike iz fajla nalozi.json. Vraca dve vrednosti, listu SVIH korisnika
* i mapu AKTIVNIH korisnika, gde je kljuc korisnicko ime.
 */
func getUsers() (map[string]*userT, []*userT) {
	var users []*userT
	file := openFile("nalozi.json")
	defer file.Close()
	d := json.NewDecoder(file)
	if err := d.Decode(&users); err != nil {
		panic("nalozi nisu dekodirani")
	}
	usersMap := make(map[string]*userT)
	for _, user := range users {
		if !user.Active {
			continue
		}
		usersMap[user.Username] = user
	}
	return usersMap, users
}

/*
* Dodaje postove u niz userMetaT::posts i vraca listu svih postova.
* Ukoliko mapa ne sadrzi autora posta, post se pridruzuje prvom korisniku koji ima bar jedan pridruzen post.
* Ukoliko ne postoji korisnik sa bar jednim pridruzenim postom, bira se (nasumicno) korisnik iz mape.
 */
func addPostMeta(users map[string]*userT) []*postMetaT {
	var posts []*postMetaT
	file := openFile("objave.json")
	defer file.Close()
	d := json.NewDecoder(file)
	if err := d.Decode(&posts); err != nil {
		panic(err.Error())
	}
	for _, post := range posts {
		var user *userT
		var ok bool
		//ako autor ne postoji u mapi aktivnih dodeli mu prvog iz mape
		if user, ok = users[post.AuthorId]; !ok {
			var prvi *userT = nil
			for _, val := range users {
				if prvi == nil {
					prvi = val
					user = prvi
				}
				if len(val.posts) > 0 {
					user = val
					break
				}
			}
		}
		user.posts = append(user.posts, post)
	}
	return posts
}

/**
* Nakon svakog ciklusa upisuje podatke nazad u json fajl. Nalozi mogu da budu deaktivirani,
* ako detektuje da su banovani ili ograniceni.
 */
func writeUsersFile(users []*userT) {
	file, err := os.OpenFile("nalozi.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	e := json.NewEncoder(file)
	e.SetIndent("", "    ")
	if err := e.Encode(users); err != nil {
		panic(err.Error())
	}
}

/**
* Upisuje meta podatke za postove. Podaci koje program menja su AuthorId i PostId.
 */
func writePostsFile(posts []*postMetaT) {
	file, err := os.OpenFile("objave.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	e := json.NewEncoder(file)
	e.SetIndent("", "    ")
	if err := e.Encode(posts); err != nil {
		panic(err.Error())
	}
}

func main() {
	var wg sync.WaitGroup
	for {
		//ucitavam ulaz iz json fajlova
		activeUsersMap, users := getUsers()
		if len(activeUsersMap) == 0 {
			panic("lista aktivnih naloga je prazna")
		}
		//upisujem postove u userT::posts niz
		posts := addPostMeta(activeUsersMap)
		//ako program nije panicio json fajlovi su ok i bekapujem ih
		backupConf("nalozi.json")
		backupConf("objave.json")
		wg.Add(len(activeUsersMap))
		log.Println("Pokrećem obradu")
		for _, user := range activeUsersMap {
			//svaki korisnik se obradjuje u posebnoj gorutini
			go func() {
				if len(user.posts) > 0 {
					user.run()
				}
				wg.Done()
			}()
		}
		//kada sve gorutine zavrse
		wg.Wait()
		//upisujem izmene u json fajlove
		log.Println("Upisujem podatke u fajlove...")
		writeUsersFile(users)
		writePostsFile(posts)
		log.Printf("Pauziram %v sekundi", RUN_TIMEOUT)
		time.Sleep(RUN_TIMEOUT * time.Second)
	}
}
