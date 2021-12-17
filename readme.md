# install sqlboiler
run this from some other directory, not within this repo (has to be outside of a go module I think)
`go install github.com/volatiletech/sqlboiler/v4@latest`

# install deps
`go install`

# build database, load data and generate models
`make models`

# run
`go run .`