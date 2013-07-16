package taskcollide

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"time"
	"fmt"
)

//paths used in the webapp
const taskRootPath = "/tasks"
const taskPostPath = taskRootPath + "/add"
const taskListPath = taskRootPath

//all handlers must be defined in init. The main() function is defined by appengine.
func init() {
	http.HandleFunc(taskRootPath, listTasks)
	http.HandleFunc(taskPostPath, postTaks)
	http.HandleFunc("/parse/", parameterTest)
}

//checks whether the user is logged in. If not, redirects to loginpage and returns nil. Otherwise, returns user object.
func checkLogin(c appengine.Context, rw http.ResponseWriter, req *http.Request) *user.User {
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, req.URL.String())
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return nil
		}
		rw.Header().Set("Location", url)
		rw.WriteHeader(http.StatusFound)
		return nil
	}
	return u
}


type Task struct {
	Owner   string
	Type    string
	Content string
	Date    time.Time
	Created time.Time
}


var listTasksTemplate = template.Must(template.New("book").Parse(listTasksTemplateHTML))

const listTasksTemplateHTML = `
<html>
  <body>
  	<table border="1">
  		<tr>
  		<th>Type</th>
  		<th>Date</th>
  		<th>Created</th>
  		<th>Topic<th>
  		</tr>
    {{range .}}
    	<tr>
    	<td>{{.Type}}</td>
    	<td>{{.Date.Format "Jan 2, 2006 at 3:04pm (MST)" }}</td>
    	<td>{{.Created.Format "01/02/2006 at 3:04pm (GMT)"}}</td>
    	<td>{{.Content}}</td>
    	</tr>
    {{end}}
    </table>
    <hr>
    <form action="` + taskPostPath + `" method="post">
      <div>
		Event type : 
		<input type="radio" name="type" value="geek" checked >Geek
		<input type="radio" name="type" value="nerd">Nerd<br>
      </div>
      <div>Topic : <textarea name="content" rows="3" cols="40"></textarea></div>
      <div>Date : <input type="text" name="date"></input></div>      
      <div><input type="submit" value="Add task"></div>
    </form>
  </body>
</html>
`

func listTasks(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	user := checkLogin(c, rw, req)
	if user == nil {
		return
	}
	q := datastore.NewQuery("Task").Filter("Owner=", user.String()).Order("-Date")
	tasks := make([]Task, 0, 10)
	if _, err := q.GetAll(c, &tasks); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := listTasksTemplate.Execute(rw, tasks); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

var tasktypeValidator = regexp.MustCompile("^geek|nerd$")

func parseTime(datestring string) (time.Time, error) {
	//try different formats
	date, err := time.Parse("1.2.2006", datestring)
	if err == nil {
		return date, nil
	}
	return time.Parse("1/2/2006", datestring)
}

func postTaks(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	user := checkLogin(c, rw, req)
	if user == nil {
		return
	}
	taskType := req.FormValue("type")
	if !tasktypeValidator.MatchString(taskType) {
		http.NotFound(rw, req)
		return
	}
	//TODO validate the content
	content := req.FormValue("content")
	//TODO parse the date from the form
	date, err := parseTime(req.FormValue("date"))
	if err != nil {
		http.NotFound(rw, req)
		log.Println("parsing date failed", req.FormValue("date"), err)
		return
	}
	task := Task{
		Owner:   user.String(),
		Type:    taskType,
		Content: content,
		Date:    date,
		Created: time.Now()}
	_, err = datastore.Put(c, datastore.NewIncompleteKey(c, "Task", nil), &task)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(rw, req, taskListPath, http.StatusFound)
}


func parameterTest(rw http.ResponseWriter, req *http.Request) {
	user := req.URL.Query().Get("user")
	fmt.Fprintf(rw, `
<html>
  <body>
  	Hello <b>%v</b>
  </body>
</html>
`, user)
}



