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
)

//paths used in the webapp
const taskRootPath = "/tasks"
const taskPostPath = taskRootPath + "/add"
const taskListPath = taskRootPath

//all handlers must be defined in init. The main() function is defined by appengine.
func init() {
	http.HandleFunc(taskRootPath, listTasks)
	http.HandleFunc(taskPostPath, postTaks)
	http.HandleFunc("/parse/", parameterTest())
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

//this template is immutable and can thus be global. Alternatively, it could be put in a closure (see example below)
//The function is created to avoid having the listTasksTemplateHTML variable avialable in the whole package
var listTasksTemplate = func() *template.Template {
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
	return template.Must(template.New("tasklist").Parse(listTasksTemplateHTML))
}()

//lists the tasks owned by the logged in user
func listTasks(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	user := checkLogin(c, rw, req)
	if user == nil {
		//the redirect has been set/send already. Nothing to do any more
		return
	}
	q := datastore.NewQuery("Task").Filter("Owner=", user.String()).Order("-Date")
	//a slice ('list') with size 0 and an initial capacity of 10
	//make is like new, but used for built-in types like lists, maps and channels
	//this is the list which will be populated with results from the query
	tasks := make([]Task, 0, 10)
	//get the tasks from the database
	if _, err := q.GetAll(c, &tasks); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	//execute the template with the tasks
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

//processes the post form. Redirects then back to the listing
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

//some experimenting: use of a closure to hide the template completely and parsing of url parameters
//This function returns itselves a function which can be used as a handler
func parameterTest() func(http.ResponseWriter, *http.Request) {
	const templateString = `<html>
			  <body>
			  	Hello <b>{{.}}</b>
			  </body>
			</html>
			`
	theTemplate := template.Must(template.New("HelloYou").Parse(templateString))

	return func(rw http.ResponseWriter, req *http.Request) {
		user := req.URL.Query().Get("user")
		if err := theTemplate.Execute(rw, user); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	}
}
