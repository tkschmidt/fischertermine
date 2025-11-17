# Understanding JavaServer Faces (JSF) and Web Scraping

## Table of Contents
1. [What is JSF?](#what-is-jsf)
2. [JSF Conversation Flows](#jsf-conversation-flows)
3. [The Execution Parameter](#the-execution-parameter)
4. [ViewState: JSF's Memory](#viewstate-jsfs-memory)
5. [Form Submission in JSF](#form-submission-in-jsf)
6. [The Bavarian Fishing Exam Website](#the-bavarian-fishing-exam-website)
7. [Why Simple Scraping Fails](#why-simple-scraping-fails)
8. [Our Solution: Fresh Sessions](#our-solution-fresh-sessions)
9. [Common Pitfalls](#common-pitfalls)

---

## What is JSF?

**JavaServer Faces (JSF)** is a server-side Java framework for building web applications. Unlike traditional stateless web applications, JSF maintains **server-side state** for user interactions.

### Key Characteristics:

- **Stateful**: The server remembers where you are in a multi-step process
- **Component-based**: UI is built from reusable components (buttons, forms, tables)
- **Event-driven**: User actions trigger server-side events
- **Session-dependent**: State is tied to your HTTP session (cookies)

### Traditional Web vs JSF:

```
Traditional Web (Stateless):
┌─────────────────────────────────────────┐
│ Client                    Server        │
├─────────────────────────────────────────┤
│ GET /search              → Process      │
│                          ← HTML page    │
│                                         │
│ POST /details?id=5       → Process      │
│                          ← HTML page    │
└─────────────────────────────────────────┘
Each request is independent


JSF (Stateful):
┌─────────────────────────────────────────┐
│ Client                    Server        │
├─────────────────────────────────────────┤
│ GET /page?execution=e1s1 → [State: e1s1]│
│                          ← HTML         │
│                                         │
│ POST /page (with state)  → [State: e1s2]│
│                          ← HTML         │
│                                         │
│ POST /page (with state)  → [State: e2s1]│
│                          ← HTML         │
└─────────────────────────────────────────┘
Server maintains conversation state
```

---

## JSF Conversation Flows

JSF organizes user interactions into **flows** (also called conversations). A flow is a sequence of related pages/steps.

### Example: Shopping Cart Flow

```
Flow "shopping":
  e1s1 → Browse products
  e1s2 → View product details
  e2s1 → Add to cart
  e2s2 → View cart
  e3s1 → Checkout
  e3s2 → Confirm order
  e4s1 → Order complete
```

### Flow Characteristics:

1. **Linear progression**: You can't jump from e1s1 to e3s1
2. **State-dependent**: Each step expects you came from the previous step
3. **Session-scoped**: Flow state is stored in your HTTP session
4. **Server-controlled**: The server decides what comes next

---

## The Execution Parameter

The `execution` parameter in the URL is JSF's way of tracking conversation state.

### Format: `execution=eXsY`

- **`e`**: Stands for "execution" (or "event")
- **`X`**: Flow/conversation number (which major step are you in?)
- **`s`**: Sub-step within that flow (which specific page?)
- **`Y`**: Step number

### Real Example from Fishing Exam Site:

```
https://...Pruefungssuche?execution=e1s1
                                    ││││
                                    ││││
                                    ││└┴─ Step 1
                                    │└─── Sub-step marker
                                    └──── Event/Flow 1

Flow progression:
e1s1 → Homepage/Welcome
e2s1 → Search form (initial)
e9s1 → Search results (list of exams)
e2s1 → Ready to submit detail request
e2s2 → Detail view for specific exam
e3s1 → Different flow (maybe registration?)
```

### Why Different Flows?

The flow number changes when you enter a different "conversation":

- **e1sX**: Initial browsing
- **e2sX**: Search/filter operations
- **e3sX**: Viewing details
- **e9sX**: Results display

This allows the server to maintain multiple independent conversations.

---

## ViewState: JSF's Memory

JSF uses a hidden form field called `javax.faces.ViewState` to maintain component state.

### In the HTML:

```html
<form id="examForm" action="/Pruefungssuche?execution=e2s1" method="post">
    <!-- Hidden state field -->
    <input type="hidden" name="javax.faces.ViewState" value="e2s1" />

    <!-- Other hidden fields -->
    <input type="hidden" name="_csrf" value="abc123..." />
    <input type="hidden" name="examForm_SUBMIT" value="1" />

    <!-- Visible form fields -->
    <input type="text" name="examForm:searchTerm" />
    <button type="submit" name="examForm:submitButton">Search</button>
</form>
```

### What ViewState Contains:

The ViewState value (`e2s1` in this simple case, but often a long encoded string) tells the server:

1. **Which page you're on**: "You're at step e2s1"
2. **Component state**: Values of UI components
3. **Conversation ID**: Links back to server-side session data

### How It Works:

```
Browser                           Server
───────                           ──────

1. GET /page?execution=e1s1
                              → Creates state for e1s1
                                Stores in session
                              ← Returns HTML with:
                                ViewState="e1s1"

2. POST /page
   ViewState=e1s1
   button=submit
                              → Checks ViewState matches session
                                Processes event
                                Updates state → e1s2
                              ← Returns HTML with:
                                ViewState="e1s2"
```

### Critical Rule:

**You MUST submit the ViewState you received, or the server rejects your request.**

This prevents:
- Skipping steps
- Replaying old requests
- CSRF attacks

---

## Form Submission in JSF

When you click a button in a JSF form, multiple things happen:

### 1. Button Identification

Each button has a unique name identifying it:

```html
<!-- Exam row 0 -->
<button name="examList:0:viewDetails">View Details</button>

<!-- Exam row 1 -->
<button name="examList:1:viewDetails">View Details</button>

<!-- Exam row 42 -->
<button name="examList:42:viewDetails">View Details</button>
```

The number (`0`, `1`, `42`) tells JSF **which exam in the list you clicked**.

### 2. Form Data Submission

When you click a button, the browser submits:

```http
POST /Pruefungssuche?execution=e2s1
Content-Type: application/x-www-form-urlencoded

_csrf=abc123&
javax.faces.ViewState=e2s1&
examForm_SUBMIT=1&
examList:42:viewDetails=
```

### 3. Server Processing

The server:

1. **Validates ViewState**: "Is this request from the current state?"
2. **Identifies component**: "Button `examList:42:viewDetails` was clicked"
3. **Triggers event**: "Show details for exam at index 42"
4. **Updates state**: e2s1 → e2s2
5. **Renders response**: Detail page for exam 42

### 4. Response

```http
HTTP/1.1 200 OK
Location: /Pruefungssuche?execution=e2s2

<html>
  <form action="/Pruefungssuche?execution=e2s2">
    <input type="hidden" name="javax.faces.ViewState" value="e2s2" />

    <h2>Exam Details</h2>
    <p>Location: Munich IHK</p>
    <p>Room: D14</p>
    <!-- ... -->
  </form>
</html>
```

---

## The Bavarian Fishing Exam Website

Let's trace a real user journey on this website:

### Step-by-Step Flow:

```
┌──────────────────────────────────────────────────────────────┐
│ Step 1: Initial Access                                      │
├──────────────────────────────────────────────────────────────┤
│ User → GET /fprApp/                                          │
│        Server creates session, assigns session cookie        │
│        Returns homepage (execution=e1s1)                     │
└──────────────────────────────────────────────────────────────┘
                            ↓
┌──────────────────────────────────────────────────────────────┐
│ Step 2: Navigate to Exam Search                             │
├──────────────────────────────────────────────────────────────┤
│ User → GET /verwaltung/Pruefungssuche?execution=e9s1         │
│        Server loads search form with search results          │
│        Page shows table of 57 exams                          │
│        Each row has a submit button:                         │
│          - pruefungsterminList:0:pruefungsterminSelect       │
│          - pruefungsterminList:1:pruefungsterminSelect       │
│          - ... (57 buttons total)                            │
│        Form action: /Pruefungssuche?execution=e2s1           │
│        ViewState: e2s1                                       │
└──────────────────────────────────────────────────────────────┘
                            ↓
┌──────────────────────────────────────────────────────────────┐
│ Step 3: User Clicks "View Details" for Exam #5              │
├──────────────────────────────────────────────────────────────┤
│ User → POST /verwaltung/Pruefungssuche?execution=e2s1        │
│        Form data:                                            │
│          javax.faces.ViewState=e2s1                          │
│          pruefungsterminList:5:pruefungsterminSelect=        │
│          _csrf=abc123                                        │
│          (other hidden fields)                               │
│                                                              │
│        Server processes:                                     │
│          1. Validates ViewState (e2s1) ✓                     │
│          2. Identifies button: list item #5                  │
│          3. Retrieves exam data for index 5                  │
│          4. Transitions: e2s1 → e2s2                         │
│          5. Renders detail page                              │
│                                                              │
│ Server ← Returns detail page (execution=e2s2)                │
│          Shows exam at index 5:                              │
│            Location: Haus der Fischerei                      │
│            Room: Sitzungssaal, 1. OG                         │
│            Address: Maiacher Straße 60d, 90441 Nürnberg      │
│            Participants: 14/14                               │
└──────────────────────────────────────────────────────────────┘
```

### Critical Observations:

1. **Button names encode position**: `pruefungsterminList:5:...` means "exam at position 5"
2. **Form action changes**: From list (e9s1) to submit endpoint (e2s1) to result (e2s2)
3. **ViewState must match**: Submit with e2s1, get back e2s2
4. **Session-dependent**: Everything is tied to your cookie session

---

## Why Simple Scraping Fails

Let's see what happens with naive approaches:

### ❌ Attempt 1: Just GET the Detail URL

```python
# Try to access detail page directly
response = requests.get('https://...?execution=e3s5')
```

**Result**:
- Returns homepage or error page
- `e3s5` doesn't exist in your session
- Server has no context for this execution ID

**Why it fails**:
- Execution IDs are session-specific
- `e3s5` was generated for someone else's session
- Your session doesn't have the conversation state needed

---

### ❌ Attempt 2: Reuse One Session for All Details

```python
session = requests.Session()

# Get list page
session.get('https://...?execution=e9s1')

# Try to get detail for exam 1
session.post('https://...?execution=e2s1', data={
    'javax.faces.ViewState': 'e2s1',
    'pruefungsterminList:0:pruefungsterminSelect': ''
})
# ✓ Works! Now at e2s2 with details for exam 1

# Try to get detail for exam 2 (still in same session)
session.post('https://...?execution=e2s1', data={
    'javax.faces.ViewState': 'e2s1',  # Wrong! We're at e2s2 now
    'pruefungsterminList:1:pruefungsterminSelect': ''
})
# ✗ Fails! Server rejects because:
#   - We're at e2s2, not e2s1
#   - ViewState mismatch
```

**Result**:
- First detail page: ✓ Success
- Second detail page: ✗ Fails
- Server returns error or redirects to homepage

**Why it fails**:
- After viewing details, state changed: e2s1 → e2s2
- Can't "go back" to e2s1 in the same session
- JSF doesn't support "back button" behavior by default

---

### ❌ Attempt 3: Try to Trick ViewState

```python
# Get list page
response = session.get('https://...?execution=e9s1')

# Try second detail request with updated ViewState
session.post('https://...?execution=e2s2', data={
    'javax.faces.ViewState': 'e2s2',  # Try e2s2 instead
    'pruefungsterminList:1:pruefungsterminSelect': ''
})
```

**Result**:
- Still fails
- Returns to list page or shows error

**Why it fails**:
- The detail page (e2s2) doesn't have buttons in the list
- Button `pruefungsterminList:1:...` doesn't exist at state e2s2
- Server-side state doesn't match what we're trying to do

---

## Our Solution: Fresh Sessions

The key insight: **Create a new session for each exam detail request**.

### Why This Works:

```
Session 1:               Session 2:               Session 3:
────────────             ────────────             ────────────

e1s1 (new session)      e1s1 (new session)      e1s1 (new session)
  ↓                       ↓                       ↓
e9s1 (list page)        e9s1 (list page)        e9s1 (list page)
  ↓                       ↓                       ↓
e2s1 (form ready)       e2s1 (form ready)       e2s1 (form ready)
  ↓                       ↓                       ↓
Submit button 0         Submit button 1         Submit button 2
  ↓                       ↓                       ↓
e2s2 (exam 0 details)   e2s2 (exam 1 details)   e2s2 (exam 2 details)

✓ Independent           ✓ Independent           ✓ Independent
✓ Clean state           ✓ Clean state           ✓ Clean state
✓ No conflicts          ✓ No conflicts          ✓ No conflicts
```

### Implementation:

```go
for each exam in exams:
    // 1. Create fresh HTTP client with new cookie jar
    jar, _ := cookiejar.New(nil)
    client := &http.Client{Jar: jar}

    // 2. Navigate to list page (establishes session)
    resp := client.Get('https://...?execution=e9s1')
    doc := parseHTML(resp.Body)

    // 3. Find button for THIS exam in THIS session
    buttonName := findButtonForExam(doc, exam)
    // e.g., "pruefungsterminList:5:pruefungsterminSelect"

    // 4. Get form action and ViewState from THIS page
    formAction := doc.Find('form').Attr('action')
    viewState := doc.Find('input[name="javax.faces.ViewState"]').Val()

    // 5. Submit form with correct button
    resp := client.Post(formAction, formData{
        'javax.faces.ViewState': viewState,
        buttonName: '',
        // ... other hidden fields
    })

    // 6. Parse detail page
    details := parseDetailPage(resp.Body)

    // 7. Match details to exam
    exam['details'] = details
```

---

## How We Match Details to the Correct Exam

This is crucial: **How do we know the detail page matches the exam we requested?**

### The Problem:

```
List page shows:
  Row 0: Nov 22, 08:00, Haus der Fischerei - Nürnberg
  Row 1: Nov 22, 08:30, IHK Campus München
  Row 2: Nov 22, 09:00, Anton-Balster-Mittelschule

We want details for Row 1 (IHK Campus München)

But button indices might change between sessions!
```

### Our Matching Strategy:

```go
// Step 1: From initial scrape, we have exam identifiers
targetExam := {
    "date_time": "22.11.2025, 08:30",
    "location": "IHK Campus München",
    "city": "München",
    "region": "Oberbayern"
}

// Step 2: In fresh session, find matching row
func findButtonForExam(listDoc, targetExam) string {
    listDoc.Find("table tr").Each(func(i, row) {
        cells := extractCells(row)

        // Match by date_time AND location
        if cells[0] == targetExam["date_time"] &&
           cells[1] == targetExam["location"] {

            // Found it! Extract the button name
            button := row.Find("input[type=submit]")
            return button.Attr("name")
            // Returns: "pruefungsterminList:1:pruefungsterminSelect"
        }
    })
}

// Step 3: Submit that specific button
// Server responds with details for exam at position 1
// Which we know is "IHK Campus München" because we matched it
```

### Why This Works:

1. **Unique identifiers**: `date_time + location` uniquely identifies each exam
2. **Dynamic matching**: We don't assume button 0 = exam 0
3. **Resilient**: Works even if list order changes
4. **Verifiable**: We can confirm the detail page matches

### Verification:

```go
// After getting detail page, we can verify:
details := parseDetailPage(detailDoc)

// Check that venue name matches
if details["exam_venue"] contains targetExam["location"] {
    // ✓ Correct detail page
    return details
} else {
    // ✗ Something went wrong
    log.Error("Detail mismatch!")
}
```

---

## Common Pitfalls

### 1. ❌ Forgetting Session Cookies

```go
// Wrong: No cookie jar
client := &http.Client{}
client.Get("https://...?execution=e9s1")  // Gets session cookie
client.Post("https://...?execution=e2s1", ...) // Cookie not sent!
```

**Fix**: Always use a cookie jar:

```go
jar, _ := cookiejar.New(nil)
client := &http.Client{Jar: jar}
```

---

### 2. ❌ Missing Hidden Form Fields

```go
// Wrong: Only sending the button
formData := url.Values{
    "pruefungsterminList:0:select": "",
}
```

**Fix**: Send ALL hidden fields:

```go
formData := url.Values{
    "_csrf": "abc123...",
    "javax.faces.ViewState": "e2s1",
    "examForm_SUBMIT": "1",
    "pruefungsterminList:0:select": "",
}
```

---

### 3. ❌ Using Wrong Form Action

```go
// Wrong: Hardcoded action
client.Post("https://...?execution=e2s1", ...)
```

**Fix**: Extract from form:

```go
action := doc.Find("form").Attr("action")
// action = "/fprApp/verwaltung/Pruefungssuche?execution=e2s1"
client.Post(baseURL + action, ...)
```

---

### 4. ❌ Submitting Unchecked Checkboxes

```go
// Wrong: Including unchecked checkboxes
formData := url.Values{
    "examForm:wheelchair": "false",  // Should not be sent!
}
```

**Fix**: Only send checked checkboxes:

```go
checkbox := doc.Find("input[name='examForm:wheelchair']")
if _, checked := checkbox.Attr("checked"); checked {
    formData.Set("examForm:wheelchair", "true")
}
// If not checked, don't send it at all
```

---

### 5. ❌ Not Handling Redirects

```go
// Wrong: Default client follows redirects silently
client := &http.Client{}
```

**Fix**: Control redirect behavior:

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if len(via) >= 20 {
            return fmt.Errorf("too many redirects")
        }
        return nil // Allow redirect
    },
}
```

---

### 6. ❌ Parallel Requests with Shared Session

```go
// Wrong: Concurrent requests sharing one session
for exam in exams:
    go func(exam) {
        client.Post(...)  // All goroutines use same client!
    }(exam)
```

**Fix**: Fresh session per exam:

```go
for exam in exams:
    go func(exam) {
        jar, _ := cookiejar.New(nil)
        client := &http.Client{Jar: jar}  // New session
        client.Post(...)
    }(exam)
```

---

## Summary

### JSF Key Concepts:

1. **Stateful**: Server maintains conversation state
2. **Execution IDs**: Track where you are in a flow (e1s1, e2s2, etc.)
3. **ViewState**: Hidden field linking browser to server state
4. **Session-dependent**: Everything tied to HTTP session cookies
5. **Component IDs**: Buttons/forms have unique identifiers

### Scraping Strategy:

1. **Fresh sessions**: New cookie jar for each detail request
2. **Complete forms**: Submit all hidden fields
3. **Dynamic matching**: Find button by exam data, not hardcoded index
4. **Follow the flow**: Navigate same path as a real user
5. **Verify results**: Confirm detail page matches expected exam

### The Trade-off:

- **Slower**: 1 request per exam (~1 second each)
- **More reliable**: Each request is independent
- **Simpler logic**: Don't need to manage complex state transitions
- **Resilient**: Works even if JSF state machine changes

This approach treats JSF as a "black box" - we don't need to understand the entire state machine, we just reset to a known good state (fresh session) for each operation.
