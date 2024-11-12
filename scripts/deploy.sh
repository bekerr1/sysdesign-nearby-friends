#!/usr/bin/env bash

CRI=${CRI:-nerdctl}

$CRI logs mysql-container
$CRI exec -it mysql-container mysql -uroot -pmy-secret-pw mydatabase
$CRI logs redis-container
$CRI exec -it redis-container redis-cli

$CRI exec -it mysql mysql -u root -padmin -D user -e "SELECT * from users;"
