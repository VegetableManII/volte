# !/bin/sh
# cscf
echo "starting... xcscf"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation;go run ./entity/xcscf/main.go -f ./config.yml"
end tell'
sleep 3s
# pgw
echo "starting... pgw"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation;go run ./entity/pgw/main.go -f ./config.yml"
end tell'
sleep 3s
# hss
echo "starting... hss"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation;go run ./entity/hss/main.go -f ./config.yml"
end tell'
sleep 3s
# mme
echo "starting... mme"
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/volte-simulation;go run ./entity/mme/main.go -f ./config.yml"
end tell'
sleep 3s
# eNodeB
echo "starting... eNodeB"
osascript -e 'tell application "Terminal"
    do script "cd ~/Documents/GitHub/volte-simulation;go run ./entity/enodeb/main.go -f ./config.yml"
end tell'
sleep 3s
osascript -e 'tell application "Terminal" 
    do script "cd ~/Documents/GitHub/test/daily;go run ./main.go"
end tell'