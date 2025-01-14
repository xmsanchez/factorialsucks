package factorial

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"golang.org/x/net/publicsuffix"
)

const BASE_URL = "https://api.factorialhr.com"

type factorialClient struct {
	http.Client
	employee_id int
	period_id   int
	calendar    []calendarDay
	shifts      []shift
	year        int
	month       int
	clock_in    string
	clock_out   string
	today_only  bool
	until_today bool
}

type period struct {
	Id          int
	Employee_id int
	Year        int
	Month       int
}

type calendarDay struct {
	Id           string
	Day          int
	Date         string
	Is_laborable bool
	Is_leave     bool
	Leave_name   string
}

type shift struct {
	Id        int64  `json:"id"`
	Period_id int64  `json:"period_id"`
	Day       int    `json:"day"`
	Clock_in  string `json:"clock_in"`
	Clock_out string `json:"clock_out"`
	Minutes   int64  `json:"minutes"`
}

type fun func() error

func handleError(spinner *spinner.Spinner, err error) {
	if err != nil {
		spinner.Stop()
		log.Fatal(err)
	}
}

func NewFactorialClient(email, password string, year, month int, in, out string, today_only, until_today bool) *factorialClient {
	spinner := spinner.New(spinner.CharSets[14], 60*time.Millisecond)
	spinner.Start()
	c := new(factorialClient)
	c.year = year
	c.month = month
	c.clock_in = in
	c.clock_out = out
	c.today_only = today_only
	c.until_today = until_today
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, _ := cookiejar.New(&options)
	c.Client = http.Client{Jar: jar}
	spinner.Suffix = " Logging in..."
	handleError(spinner, c.login(email, password))
	spinner.Suffix = " Getting periods data..."
	handleError(spinner, c.setPeriodId())
	spinner.Suffix = " Getting calendar data..."
	handleError(spinner, c.setCalendar())
	spinner.Suffix = " Getting shifts data..."
	handleError(spinner, c.setShifts())
	spinner.Stop()
	return c
}

func (c *factorialClient) ClockIn(dry_run bool) {
	spinner := spinner.New(spinner.CharSets[14], 60*time.Millisecond)
	var t time.Time
	var message string
	var body []byte
	var shift shift
	var resp *http.Response
	var ok bool
	now := time.Now()
	shift.Period_id = int64(c.period_id)
	shift.Clock_in = c.clock_in
	shift.Clock_out = c.clock_out
	shift.Minutes = 0
	for _, d := range c.calendar {
		spinner.Restart()
		spinner.Reverse()
		t = time.Date(c.year, time.Month(c.month), d.Day, 0, 0, 0, 0, time.UTC)

		if t.Weekday() == 5 {  // if it's Friday
			shift.Clock_in = "09:00"
			shift.Clock_out = "15:00"
		} else {
			shift.Clock_in = c.clock_in
			shift.Clock_out = c.clock_out
		}

		message = fmt.Sprintf("%s... ", t.Format("02 Jan"))
		spinner.Prefix = message + " "
		clocked_in, clocked_times := c.clockedIn(d.Day, shift)
		if clocked_in {
			message = fmt.Sprintf("%s ❌ Period overlap: %s\n", message, clocked_times)
		} else if d.Is_leave {
			message = fmt.Sprintf("%s ❌ %s\n", message, d.Leave_name)
		} else if !d.Is_laborable {
			message = fmt.Sprintf("%s ❌ %s\n", message, t.Format("Monday"))
		} else if c.today_only && d.Day != now.Day() {
			message = fmt.Sprintf("%s ❌ %s\n", message, "Skipping: --today")
		} else if c.until_today && d.Day > now.Day() {
			message = fmt.Sprintf("%s ❌ %s\n", message, "Skipping: --until-today")
		} else {
			ok = true
			if !dry_run {
				ok = false
				shift.Day = d.Day
				body, _ = json.Marshal(shift)
				resp, _ = c.Post(BASE_URL+"/attendance/shifts", "application/json;charset=UTF-8", bytes.NewBuffer(body))
				if resp.StatusCode == 201 {
					ok = true
				}
			}
			if ok {
				message = fmt.Sprintf("%s ✅ %s - %s\n", message, shift.Clock_in, shift.Clock_out)
			} else {
				message = fmt.Sprintf("%s ❌ Error when attempting to clock in\n", message)
			}
		}
		spinner.Stop()
		fmt.Print(message)
	}
	fmt.Println("done!")
}

func (c *factorialClient) login(email, password string) error {
	getCSRFToken := func(resp *http.Response) string {
		data, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		start := strings.Index(string(data), "<meta name=\"csrf-token\" content=\"") + 33
		end := strings.Index(string(data)[start:], "\" />")
		return string(data)[start : start+end]
	}

	getLoginError := func(resp *http.Response) string {
		data, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		start := strings.Index(string(data), "<div class=\"flash flash--wrong\">") + 32
		if start < 0 {
			return ""
		}
		end := strings.Index(string(data)[start:], "</div>")
		if start < 0 || end-start > 100 {
			return ""
		}
		return string(data)[start : start+end]
	}

	resp, _ := c.Get(BASE_URL + "/users/sign_in")
	csrf_token := getCSRFToken(resp)
	body := url.Values{
		"authenticity_token": {csrf_token},
		"return_host":        {"factorialhr.es"},
		"user[email]":        {email},
		"user[password]":     {password},
		"user[remember_me]":  {"0"},
		"commit":             {"Sign in"},
	}
	resp, _ = c.PostForm(BASE_URL+"/users/sign_in", body)
	if err := getLoginError(resp); err != "" {
		return errors.New(err)
	}
	return nil
}

func (c *factorialClient) setPeriodId() error {
	err := errors.New("Could not find the specified year/month in the available periods (" + strconv.Itoa(c.month) + "/" + strconv.Itoa(c.year) + ")")
	resp, _ := c.Get(BASE_URL + "/attendance/periods?year=" + strconv.Itoa(c.year) + "&month=" + strconv.Itoa(c.month))
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return err
	}
	var periods []period
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &periods)
	for _, p := range periods {
		if p.Year == c.year && p.Month == c.month {
			c.employee_id = p.Employee_id
			c.period_id = p.Id
			return nil
		}
	}
	return err
}

func (c *factorialClient) setCalendar() error {
	u, _ := url.Parse(BASE_URL + "/attendance/calendar")
	q := u.Query()
	q.Set("id", strconv.Itoa(c.employee_id))
	q.Set("year", strconv.Itoa(c.year))
	q.Set("month", strconv.Itoa(c.month))
	u.RawQuery = q.Encode()
	resp, _ := c.Get(u.String())
	if resp.StatusCode != 200 {
		return errors.New("Error retrieving calendar data")
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &c.calendar)
	sort.Slice(c.calendar, func(i, j int) bool {
		return c.calendar[i].Day < c.calendar[j].Day
	})
	return nil
}

func (c *factorialClient) setShifts() error {
	u, _ := url.Parse(BASE_URL + "/attendance/shifts")
	q := u.Query()
	q.Set("employee_id", strconv.Itoa(c.employee_id))
	q.Set("year", strconv.Itoa(c.year))
	q.Set("month", strconv.Itoa(c.month))
	u.RawQuery = q.Encode()
	resp, _ := c.Get(u.String())
	if resp.StatusCode != 200 {
		return errors.New("Error retrieving shifts data")
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &c.shifts)
	return nil
}

func (c *factorialClient) clockedIn(day int, input_shift shift) (bool, string) {
	clock_in, _ := strconv.Atoi(strings.Join(strings.Split(input_shift.Clock_in, ":"), ""))
	clock_out, _ := strconv.Atoi(strings.Join(strings.Split(input_shift.Clock_out, ":"), ""))
	for _, shift := range c.shifts {
		if shift.Day == day {
			shift_clock_in, _ := strconv.Atoi(strings.Join(strings.Split(shift.Clock_in, ":"), ""))
			shift_clock_out, _ := strconv.Atoi(strings.Join(strings.Split(shift.Clock_out, ":"), ""))
			if (clock_in < shift_clock_in && shift_clock_in < clock_out) ||
				(clock_in < shift_clock_out && shift_clock_out < clock_out) ||
				(shift_clock_in <= clock_in && shift_clock_out >= clock_out) {
				return true, strings.Join([]string{shift.Clock_in, shift.Clock_out}, " - ")
			}
		}
	}
	return false, ""
}

func (c *factorialClient) ResetMonth() {
	var t time.Time
	var message string
	for _, shift := range c.shifts {
		req, _ := http.NewRequest("DELETE", BASE_URL+"/attendance/shifts/"+strconv.Itoa(int(shift.Id)), nil)
		resp, _ := c.Do(req)
		t = time.Date(c.year, time.Month(c.month), shift.Day, 0, 0, 0, 0, time.UTC)
		message = fmt.Sprintf("%s... ", t.Format("02 Jan"))
		if resp.StatusCode != 204 {
			fmt.Print(fmt.Sprintf("%s ❌ Error when attempting to delete shift: %s - %s\n", message, shift.Clock_in, shift.Clock_out))
		} else {
			fmt.Print(fmt.Sprintf("%s ✅ Shift deleted: %s - %s\n", message, shift.Clock_in, shift.Clock_out))
		}
		defer resp.Body.Close()
	}
	fmt.Println("done!")
}
