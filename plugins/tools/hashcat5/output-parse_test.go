package hashcat5

import (
	"fmt"
	"strings"
	"testing"
)

const TestStatus1 = `
hashcat (v3.00-1-g67a8d97) starting...

Hashes: 1 hashes; 1 unique digests, 1 unique salts
Bitmaps: 16 bits, 65536 entries, 0x0000ffff mask, 262144 bytes, 5/13 rotates
Rules: 100000
Applicable Optimizers:
* Zero-Byte
* Single-Hash
* Single-Salt
Watchdog: Temperature abort trigger set to 90c
Watchdog: Temperature retain trigger set to 75c

WARNING: Failed to set initial fan speed for device #1
WARNING: Failed to set initial fan speed for device #2
WARNING: Failed to set initial fan speed for device #3
WARNING: Failed to set initial fan speed for device #4
WARNING: Failed to set initial fan speed for device #5
WARNING: Failed to set initial fan speed for device #6
WARNING: Failed to set initial fan speed for device #7
WARNING: Failed to set initial fan speed for device #8
Generated dictionary stats for example.dict: 1080240 bytes, 129988 words, 12998800000 keyspace

ATTENTION!
  The wordlist or mask you are using is too small.
  Therefore, hashcat is unable to utilize the full parallelization power of your device(s).
  The cracking speed will drop.
  Workaround: https://hashcat.net/wiki/doku.php?id=frequently_asked_questions#how_to_create_more_work_for_full_speed

INFO: approaching final keyspace, workload adjusted


STATUS  2       SPEED   16144   4.034016        16236   4.025438        16249   3.775047        16190   4.152047        16156   4.043844        16144   4.021375        16194   3.794531        16190   3.785211        EXEC_RUNTIME    0.266413     0.265526        0.379323        0.274337        0.268150        0.265876        0.384700        0.384268        CURKU   0       PROGRESS        138307048       12998800000     RECHASH 0       1       RECSALT 0       1       TEMP48       47      48      45      47      47      45      46


STATUS  2       SPEED   16179   4.039344        16249   4.064578        16230   3.792484        16190   4.178555        16213   4.067875        16249   4.050852        16230   3.818547        16245   3.792273        EXEC_RUNTIME    0.266640     0.265554        0.385833        0.276971        0.271365        0.265707        0.385839        0.384649        CURKU   0       PROGRESS        301912949       12998800000     RECHASH 0       1       RECSALT 0       1       TEMP50       48      50      46      49      49      46      48


STATUS  2       SPEED   16202   4.028039        16179   4.024180        16230   3.787477        16213   4.151969        16213   4.060516        16190   4.023180        16176   3.816797        16245   3.781398        EXEC_RUNTIME    0.263835     0.267435        0.385435        0.281666        0.271415        0.262816        0.387224        0.385313        CURKU   0       PROGRESS        465730075       12998800000     RECHASH 0       1       RECSALT 0       1       TEMP51       50      51      48      50      50      48      49


STATUS  2       SPEED   16156   4.044242        16247   4.058719        16248   3.789883        16202   4.166594        16133   4.052352        16213   4.050828        16249   3.827891        16136   3.776227        EXEC_RUNTIME    0.264790     0.264143        0.378840        0.276282        0.262948        0.266951        0.385018        0.382662        CURKU   0       PROGRESS        628946008       12998800000     RECHASH 0       1       RECSALT 0       1       TEMP52       51      52      49      52      52      50      51


STATUS  2       SPEED   16167   4.045203        16156   4.082281        16249   3.787258        16156   4.232250        16224   4.080820        16249   4.076391        16249   3.833641        16208   3.780180        EXEC_RUNTIME    0.266355     0.262116        0.382170        0.278408        0.267949        0.265691        0.385887        0.382876        CURKU   0       PROGRESS        791999451       12998800000     RECHASH 0       1       RECSALT 0       1       TEMP53       52      53      50      53      53      51      52


STATUS  2       SPEED   16156   4.056211        16156   4.013547        16230   3.787437        16156   4.155086        16249   4.089844        16224   4.065961        16249   3.821719        16136   3.781164        EXEC_RUNTIME    0.267455     0.258008        0.375399        0.271955        0.268523        0.268422        0.386849        0.373778        CURKU   0       PROGRESS        955377866       12998800000     RECHASH 0       1       RECSALT 0       1       TEMP54       53      54      51      54      54      52      53


`

var TestStatus2 = `
hashcat (v3.00-1-g67a8d97) starting...

Hashes: 32 hashes; 32 unique digests, 1 unique salts
Bitmaps: 16 bits, 65536 entries, 0x0000ffff mask, 262144 bytes, 5/13 rotates
Rules: 1000000
Applicable Optimizers:
* Zero-Byte
* Precompute-Init
* Precompute-Merkle-Demgard
* Meet-In-The-Middle
* Early-Skip
* Not-Salted
* Not-Iterated
* Single-Salt
* Raw-Hash
Watchdog: Temperature abort trigger set to 90c
Watchdog: Temperature retain trigger set to 75c

WARNING: Failed to set initial fan speed for device #1
WARNING: Failed to set initial fan speed for device #2
WARNING: Failed to set initial fan speed for device #3
WARNING: Failed to set initial fan speed for device #4
WARNING: Failed to set initial fan speed for device #5
WARNING: Failed to set initial fan speed for device #6
WARNING: Failed to set initial fan speed for device #7
WARNING: Failed to set initial fan speed for device #8
Generated dictionary stats for /usr/local/hashcat/dicts/crackstation-human-only.txt: 716441107 bytes, 63941069 words, 63768655000000 keyspace


STATUS  2       SPEED   9417408 4.088781        9101664 3.961703        9225216 3.730328        8588800 3.674992        9799680 3.839867        9012432 3.860656        8837400 3.823297        8940800 3.661930        EXEC_RUNTIME    18.567990    19.389148       18.251862       18.708600       16.271819       18.893733       19.460506       18.638150       CURKU   0       PROGRESS        9968195200      63768655000000  RECHASH 0       32      RECSALT 0       1       TEMP40       38      39      37      40      39      36      37


STATUS  2       SPEED   27579552        11.881734       27304992        11.739398       28740096        11.592508       26797056        11.479484       30052352        11.661766       27730560        11.753055       27219192        11.685914    28252928        11.586180       EXEC_RUNTIME    18.431376       19.156688       18.206796       18.731019       16.115295       18.695493       19.314854       18.663070       CURKU   0       PROGRESS        29264621184     63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    40      39      40      38      40      40      37      38


STATUS  2       SPEED   43051008        18.418859       44808192        19.099867       45416448        18.171859       43974656        18.669930       41811968        16.045492       44368896        18.654242       45247488        19.256742    45776896        18.630859       EXEC_RUNTIME    18.308156       18.994184       18.060585       18.563451       15.936598       18.546308       19.144083       18.521917       CURKU   0       PROGRESS        48915108480     63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    41      39      41      38      41      40      37      38


STATUS  2       SPEED   43051008        18.373242       44808192        19.111031       45416448        18.064469       43974656        18.612414       41811968        16.008555       44368896        18.638578       45247488        19.242891    45776896        18.563547       EXEC_RUNTIME    18.262604       19.005019       17.953953       18.506325       15.899574       18.524433       19.130247       18.454564       CURKU   0       PROGRESS        68300925568     63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    41      40      41      39      41      41      38      39


STATUS  2       SPEED   43051008        18.626836       44808192        19.393039       45416448        18.193711       43974656        18.663914       41811968        16.290398       44368896        18.876930       45247488        19.505672    45776896        18.630984       EXEC_RUNTIME    18.514243       19.285492       18.084920       18.551907       16.177199       18.758253       19.368090       18.521463       CURKU   0       PROGRESS        87330360960     63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    42      40      42      39      42      41      38      39


STATUS  2       SPEED   43051008        18.646211       44808192        19.411711       45416448        18.391414       43974656        18.952562       41811968        16.372305       44368896        18.880531       45247488        19.544523    45776896        18.871070       EXEC_RUNTIME    18.534103       19.304226       18.284192       18.840975       16.258531       18.762580       19.410217       18.761614       CURKU   0       PROGRESS        106491618944    63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    42      41      42      39      42      41      38      40


STATUS  2       SPEED   43051008        18.690320       44808192        19.361766       45416448        18.362938       43974656        19.021359       41811968        16.418570       44368896        18.926250       45247488        19.577078    45776896        18.924914       EXEC_RUNTIME    18.580723       19.255133       18.256674       18.904434       16.309196       18.819049       19.443897       18.816123       CURKU   0       PROGRESS        125655422592    63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    42      41      43      40      43      42      39      40


STATUS  2       SPEED   43051008        18.539664       44808192        19.288977       45416448        18.417156       43974656        19.040016       41811968        16.254156       44368896        18.813234       45247488        19.432594    45776896        18.958727       EXEC_RUNTIME    18.430269       19.181949       18.311218       18.922316       16.144642       18.705907       19.324197       18.849629       CURKU   0       PROGRESS        144946351744    63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    43      41      43      40      43      42      39      40


STATUS  2       SPEED   43051008        18.426750       44808192        19.252992       45416448        18.319414       43974656        18.901734       41811968        16.114492       44368896        18.725461       45247488        19.384836    45776896        18.821578       EXEC_RUNTIME    18.316841       19.129454       18.213479       18.783535       16.003154       18.616421       19.275897       18.711729       CURKU   0       PROGRESS        164242383488    63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    43      42      43      41      43      43      40      41


STATUS  6       SPEED   43051008        18.469703       44808192        19.140461       45416448        18.180828       43974656        18.808000       41811968        16.157648       44368896        18.739242       45247488        19.248437    45776896        18.746070       EXEC_RUNTIME    18.353227       19.010978       18.054484       18.713342       16.040953       18.626292       19.143381       18.615106       CURKU   0       PROGRESS        178894234240    63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    43      42      44      41      43      43      40      41

Started: Tue Jul  5 00:27:34 2016
Stopped: Tue Jul  5 00:28:48 2016
root@sb-gpu-02:/jmmcatee# ./hashcat-3.00/hashcat64.bin -m 1000 -g 1000000 hashes.txt /usr/local/hashcat/dicts/crackstation-human-only.txt --status --status-timer=30 --machine-readable
hashcat (v3.00-1-g67a8d97) starting...

Hashes: 32 hashes; 32 unique digests, 1 unique salts
Bitmaps: 16 bits, 65536 entries, 0x0000ffff mask, 262144 bytes, 5/13 rotates
Rules: 1000000
Applicable Optimizers:
* Zero-Byte
* Precompute-Init
* Precompute-Merkle-Demgard
* Meet-In-The-Middle
* Early-Skip
* Not-Salted
* Not-Iterated
* Single-Salt
* Raw-Hash
Watchdog: Temperature abort trigger set to 90c
Watchdog: Temperature retain trigger set to 75c

WARNING: Failed to set initial fan speed for device #1
WARNING: Failed to set initial fan speed for device #2
WARNING: Failed to set initial fan speed for device #3
WARNING: Failed to set initial fan speed for device #4
WARNING: Failed to set initial fan speed for device #5
WARNING: Failed to set initial fan speed for device #6
WARNING: Failed to set initial fan speed for device #7
WARNING: Failed to set initial fan speed for device #8
Cache-hit dictionary stats /usr/local/hashcat/dicts/crackstation-human-only.txt: 716441107 bytes, 63768655 words, 63768655000000 keyspace


STATUS  2       SPEED   43084800        18.618758       42893312        17.844445       39975936        16.761172       41811968        17.648328       42611712        16.858242       40212480        17.209937       43563520        18.871969    43490304        18.259414       EXEC_RUNTIME    18.467385       17.734428       16.652587       17.533034       16.745900       17.102569       18.755732       18.149847       CURKU   0       PROGRESS        549889370240    63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    49      47      48      46      49      48      45      46


STATUS  2       SPEED   43084800        18.438922       42893312        17.794414       39975936        16.665945       41811968        17.501562       42611712        16.715781       40212480        17.344133       43563520        18.823141    43490304        18.357992       EXEC_RUNTIME    18.320157       17.656613       16.555207       17.393743       16.605621       17.211420       18.713816       18.220711       CURKU   0       PROGRESS        1123760315008   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    53      52      52      50      53      52      50      51


STATUS  2       SPEED   43084800        18.611930       42893312        17.491008       39975936        16.923641       41811968        17.267219       42611712        17.056156       40212480        17.255352       43563520        18.945391    43490304        18.421680       EXEC_RUNTIME    18.496046       17.379510       16.811325       17.155230       16.943125       17.146715       18.835367       18.268788       CURKU   0       PROGRESS        1696874600576   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    55      54      55      52      56      55      53      54


STATUS  2       SPEED   43084800        18.587414       42893312        17.562016       39975936        16.802789       41811968        17.457320       42611712        16.871969       40212480        17.266250       43563520        18.868125    43490304        18.698461       EXEC_RUNTIME    18.447517       17.451658       16.694671       17.348420       16.763969       17.158235       18.755836       18.588409       CURKU   0       PROGRESS        2269131233920   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    56      55      56      54      57      56      54      55


STATUS  2       SPEED   43084800        18.512063       42893312        17.577852       39975936        17.054937       41811968        17.460578       42611712        17.007242       40212480        17.375781       43563520        18.749977    43490304        18.467273       EXEC_RUNTIME    18.400890       17.467446       16.944536       17.349865       16.896248       17.267024       18.640704       18.356582       CURKU   0       PROGRESS        2841705827456   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      56      56      55      58      56      55      56


STATUS  2       SPEED   43084800        18.568977       42893312        17.898094       39975936        16.557672       41811968        17.586430       42611712        16.692266       40212480        17.354305       43563520        18.825313    43490304        18.272094       EXEC_RUNTIME    18.458186       17.784091       16.447302       17.475701       16.580889       17.205312       18.716808       18.155928       CURKU   0       PROGRESS        3414290693760   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      57      57      55      58      57      55      57


STATUS  2       SPEED   43084800        18.530211       42893312        17.903844       39975936        16.840703       41811968        17.309047       42611712        16.787672       40212480        17.561406       43563520        18.632703    43490304        18.604703       EXEC_RUNTIME    18.418318       17.791519       16.723742       17.200798       16.677601       17.452009       18.518594       18.495780       CURKU   0       PROGRESS        3986232886016   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    58      57      57      55      59      57      56      57


STATUS  2       SPEED   43084800        21.928242       42893312        17.691844       39975936        17.267953       41811968        17.331891       42611712        21.250828       40212480        17.514180       43563520        21.793867    43490304        21.630570       EXEC_RUNTIME    21.817378       17.556684       17.130259       17.183345       21.142223       17.404346       21.679051       21.521932       CURKU   0       PROGRESS        4555201414400   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      57      57      56      58      57      55      57


STATUS  2       SPEED   43084800        21.836187       42893312        17.759117       39975936        17.014359       41811968        17.454250       42611712        21.550359       40212480        17.565477       43563520        21.829648    43490304        21.399875       EXEC_RUNTIME    21.703425       17.643058       16.905528       17.344502       21.440252       17.442452       21.714394       21.289814       CURKU   0       PROGRESS        5079533647616   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      57      57      56      58      57      55      57


STATUS  2       SPEED   43084800        21.766922       42893312        20.534141       39975936        17.097281       41811968        21.040430       42611712        21.249398       40212480        17.649469       43563520        22.194344    43490304        21.376531       EXEC_RUNTIME    21.660292       20.426638       16.962895       20.933948       21.141559       17.539718       22.076016       21.266385       CURKU   4432541 PROGRESS        5598722915904   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      57      57      55      58      57      55      57


STATUS  2       SPEED   43084800        21.366289       42893312        20.692664       39975936        17.248023       41811968        20.969344       42611712        21.568211       40212480        17.572359       43563520        22.017820    43490304        21.480031       EXEC_RUNTIME    21.257953       20.540651       17.094445       20.842467       21.444663       17.425550       21.889188       21.369202       CURKU   4432541 PROGRESS        6100439751744   63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      57      57      55      58      57      55      57


`

var TestStatus3 = `
STATUS  2       SPEED   43084800        22.503773       42893312        20.792844       39975936        19.658547       41811968        20.861344       42611712        21.318234       40212480        20.094789       43563520        22.118852    43490304        22.223383       EXEC_RUNTIME    22.393160       20.683908       19.553874       20.750683       21.209579       19.987781       22.000827       22.112880       CURKU   8227146 PROGRESS        15859234245376  63768655000000       RECHASH 0       32      RECSALT 0       1       TEMP    57      56      56      55      58      56      55      57


STATUS  2       SPEED   43084800        22.644039       42893312        20.805937       39975936        19.397805       41811968        20.674008       42611712        21.131867       40212480        20.177867       43563520        22.193227    43490304        22.047813       EXEC_RUNTIME    22.533905       20.698997       19.291950       20.564572       21.024320       20.032417       22.079528       21.940164       CURKU   10412117        PROGRESS        16350422667328       63768655000000  RECHASH 0       32      RECSALT 0       1       TEMP    57      56      56      55      58      57      55      56


STATUS  2       SPEED   43084800        22.360141       42893312        20.894945       39975936        19.627109       41811968        20.870742       42611712        21.347758       40212480        20.363867       43563520        22.037266    43490304        22.053039       EXEC_RUNTIME    22.249551       20.790238       19.498310       20.749272       21.239166       20.249736       21.929832       21.933542       CURKU   10412117        PROGRESS        16831225065536       63768655000000  RECHASH 0       32      RECSALT 0       1       TEMP    57      56      56      55      58      56      55      57

`

var TestStatus4 = `

WARNING: Failed to set initial fan speed for device #1
WARNING: Failed to set initial fan speed for device #2
WARNING: Failed to set initial fan speed for device #3
WARNING: Failed to set initial fan speed for device #4
WARNING: Failed to set initial fan speed for device #5
WARNING: Failed to set initial fan speed for device #6
WARNING: Failed to set initial fan speed for device #7
WARNING: Failed to set initial fan speed for device #8
STATUS  2       SPEED   177376628       12.396055       161669552       11.217672       159502200       11.032617       186706872       13.127570       178122648       12.438063       161669552       11.296672       176691944       12.327398       178122648       12.286656       EXEC_RUNTIME    12.057285       10.865864       10.695928       12.763795       12.080824       10.940713       11.968389       11.926803    CURKU   1236066304      PROGRESS        11297119910912  6403748062057711647     RECHASH 0       32      RECSALT 0       1       TEMP    70      71      70      71      72      70      68      69


`

var TestStatus5 = ``

var TestStatus6 = `STATUS	2	SPEED	200116	2.474344	EXEC_RUNTIME	2.010319	CURKU	376323	PROGRESS	1550333385	58592364160	RECHASH	4	5	RECSALT	0	1	TEMP	55	
`

func TestParseStatus1(t *testing.T) {
	status, _ := ParseMachineOutput(TestStatus1)
	fmt.Printf("%+v\n", status)
}

func TestParseStatus2(t *testing.T) {
	status, _ := ParseMachineOutput(TestStatus2)
	fmt.Printf("%+v\n", status)
}

func TestParseStatus3(t *testing.T) {
	status, _ := ParseMachineOutput(TestStatus3)
	fmt.Printf("%+v\n", status)
}

func TestParseStatus4(t *testing.T) {
	status, _ := ParseMachineOutput(TestStatus4)
	fmt.Printf("%+v\n", status)
}

func TestParseStatus5(t *testing.T) {
	status, _ := ParseMachineOutput(TestStatus5)
	fmt.Printf("%+v\n", status)
}

func TestParseStatus6(t *testing.T) {
	status, _ := ParseMachineOutput(TestStatus6)
	fmt.Printf("%+v\n", status)
}

var PotFileContent_1 = `7C77EE05A297638DFF9D75B7C28561E3:PQLSDJK
F96946077DBF98C3F45D1D4576EAFE23:PQLSDJK1234
858B5E5FE5DF0ABD69F2EE9A8B55385B:PQLSDJK567
153B57A9B699C4807F721249AFA492FC:PQLSDJK000
DFEF8C46EFBA5C30685B2B6FDA97A8B3:PQLSDJK!@#$`

func TestParseShowPotFile_1(t *testing.T) {
	r := strings.NewReader(PotFileContent_1)

	// leftSplit is 0 because the input would be just an NTLM hash in this instance
	count, hashes := ParseShowPotFile(r, 0, "")
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Hashes:\n")
	for i := range hashes {
		fmt.Printf("\t%v\n", hashes[i])
	}
}

var PotFileContent_2 = `342c0cf1ddcad9f3:user:PASSWORD
2d7448670460a07c:user3:CHANGEME
29545932AFD2F0C7:USER5:CHANGEME:::`

func TestParseShowPotFile_2(t *testing.T) {
	r := strings.NewReader(PotFileContent_2)

	// leftSplit is 1 because the input would be an Oracle hash of hash:salt
	count, hashes := ParseShowPotFile(r, 1, "")
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Hashes:\n")
	for i := range hashes {
		fmt.Printf("\t%v\n", hashes[i])
	}
}

var PotFileContent_3 = `user1:500:E52CAC67419A9A224A3B108F3FA6CB6D:A4F49C406510BDCAB6824EE7C30FD852::::Password
user2:5001:E52CAC67419A9A229FE041B7DC3D21A4:4882BEABE01BE6928C646251A961DC16::::PasswordTwo
user3:2494:048ADC2C7965C60F02657A8D8EF025E2:DA3AD41053E2A0AD425920F2ADBB000B::::Pass Word
domain\user4:1000:2246C8E4347FF4480B4F952384D3CD46:28D39B540448527F8936EB5044AB9126::::CHANGE?::ME
domain\user4:1000:2246C8E4347FF4480B4F952384D3CD46:28D39B540448527F8936EB5044AB9126:::::CHANGE?::ME
domain2\user5:3993:2246C8E4347FF4483B41676011483133:E2C0FC226D7D471C06506607CF35F1C9::::CHANGE?::
user6:42043:136405D7E4EAA9CB7CF869C681AFAF70:2C515086DC8C8ADB677FF6343B6BB46A::::GREEnnn#$(@?
user7:21300:4E874789408A6598AAD3B435B51404EE:A6B6BAFA62D7DFC05053783DDAD6FA49::::dfjajef`

func TestParseShowPotFile_3(t *testing.T) {
	r := strings.NewReader(PotFileContent_3)

	// leftSplit is 6 because the input would be an PWDUMP format file
	count, hashes := ParseShowPotFile(r, 6, "")
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Hashes:\n")
	for i := range hashes {
		fmt.Printf("\t%v\n", hashes[i])
	}
}

var PotFileContent_4 = `A4F49C406510BDCAB6824EE7C30FD852:Password
4882BEABE01BE6928C646251A961DC16:PasswordTwo
DA3AD41053E2A0AD425920F2ADBB000B:Pass Word
28D39B540448527F8936EB5044AB9126:CHANGE?::ME
E2C0FC226D7D471C06506607CF35F1C9:CHANGE?::
E2C0FC226D7D471C06506607CF35F1C9::CHANGE?::
2C515086DC8C8ADB677FF6343B6BB46A:GREEnnn#$(@?
A6B6BAFA62D7DFC05053783DDAD6FA49:dfjajef`

func TestParseShowPotFile_4a(t *testing.T) {
	r := strings.NewReader(PotFileContent_4)

	// leftSplit is 6 because the input would be an PWDUMP format file
	count, hashes := ParseHashcatOutputFile(r, 6, "1000")
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Hashes:\n")
	for i := range hashes {
		fmt.Printf("\t%v\n", hashes[i])
	}
}

func TestParseShowPotFile_4b(t *testing.T) {
	r := strings.NewReader(PotFileContent_4)

	// leftSplit is 6 because the input would be an PWDUMP format file
	count, hashes := ParseHashcatOutputFile(r, 6, "3000")
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Hashes:\n")
	for i := range hashes {
		fmt.Printf("\t%v\n", hashes[i])
	}
}

func TestParseShowPotFile_4c(t *testing.T) {
	r := strings.NewReader(PotFileContent_4)

	// leftSplit is 0 because the input would be NTLM only
	count, hashes := ParseHashcatOutputFile(r, 0, "1000")
	fmt.Printf("Count: %d\n", count)
	fmt.Printf("Hashes:\n")
	for i := range hashes {
		fmt.Printf("\t%v\n", hashes[i])
	}
}
