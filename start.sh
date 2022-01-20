# !/bin/sh
# cscf
echo "starting... xcscf"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation/entity;go run ./xcscf/main.go -f ./config.yml"
end tell'
sleep 3s
# pgw
echo "starting... pgw"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation/entity;go run ./pgw/main.go -f ./config.yml"
end tell'
sleep 3s
# hss
echo "starting... hss"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation/entity;go run ./hss/main.go -f ./config.yml"
end tell'
sleep 3s
# mme
echo "starting... mme"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation/entity;go run ./mme/main.go -f ./config.yml"
end tell'
sleep 3s
# eNodeB
echo "starting... eNodeB"
osascript -e 'tell application "Terminal"
    do script "cd ~/Documents/GitHub/volte-simulation/entity;go run ./enodeb/main.go -f ./config.yml"
end tell'
sleep 3s
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/test/daily;go run ./main.go"
end tell'