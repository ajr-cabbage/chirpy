### chirpy
A boot.dev guided project for learning HTTP servers.
#### Requirements
- golang
- postgresql
- goose
#### Endpoints
###### "/app/"
- The "homepage"
###### "/admin
- "GET /metrics": site hit counter
- "POST /reset": manual reset of the database
###### "/api/"
- "POST /users": Create user
- "POST /chirps": Create a "chirp"
- "GET /chirps": Retrieve chirps. Optional "author_id" and "sort" queries.
- "GET /chirps{chirpID}": Retrieve a specific chirp
- "POST /login": Login with user. Get access and refresh token.
- "POST /refresh": Refresh access token
- "POST /revoke": Revoke refresh token
- "PUT /users": Update email/password
- "DELETE /chirps/{chirpID}": Remove a chirp
