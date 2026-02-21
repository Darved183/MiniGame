# Сборка без проверки VCS (устраняет ошибку "error obtaining VCS status: exit status 128")
go build -buildvcs=false -o game.exe .
