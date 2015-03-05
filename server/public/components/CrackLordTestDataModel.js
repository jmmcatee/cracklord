angular.module('cracklord').service('UserDataModel', function UserDataModel($filter) {
    this.data = {
        "readonly": { "password": "readonly", "role": "read-only" },
        "user": { "password": "user", "role": "standard user" },
        "admin": { "password": "admin", "role": "administrator" }
    }

    this.newid = function() {
        var text = "";
        var possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

        for( var i=0; i < 12; i++ ) {
            text += possible.charAt(Math.floor(Math.random() * possible.length));
        }

        return text;
    }

    this.login = function(user, pass) {
        var found = this.data[user];
        if(found) {
            if(found["password"] === pass) {
                return {
                    "role": found["role"],
                    "token": this.newid()
                };
            } else {
                return false;
            }
        } else {
            return false;
        }
    }
});

angular.module('cracklord').service('ToolsDataModel', function ToolsDataModel($filter) {
    this.data = [
        { "toolid": "63ee8045-966f-449e-9839-58e7e0586f3c", "name": "Hashcat", "version": "1.3.3", "form": [ "algorithm", "dictionary", "rules", { "key": "hashes", "type": "textarea", "placeholder": "Hashes go here!" } ], "schema": { "type": "object", "properties": { "algorithm": { "title": "Algorithm", "type": "string", "enum": [ "NTLM", "NTLMv2", "ms-cache", "ms-cache v2", "SQL 2005", "SQL 2008", "MD5" ] }, "dictionary": { "title": "Dictionary", "type": "string", "enum": [ "crackstation", "crackstation-human-only", "m3g9tr0n", "words-english" ] }, "rules": { "title": "Rule File", "type": "string", "enum": [ "d3ad0ne", "t0xic" ] }, "hashes": { "title": "Hashes", "type": "string", "description": "Note: Use the file format as required by hashcat" } } } },
        { "toolid": "8d660ce9-f15d-40a3-a997-a4e8867cb802", "name": "John the Ripper", "version": "1.7.9", "form": [ "algorithm", "dictionary", "rules", { "key": "hashes", "type": "textarea", "placeholder": "Hashes go here!" } ], "schema": { "type": "object", "properties": { "algorithm": { "title": "Algorithm", "type": "string", "enum": [ "NTLM", "NTLMv2", "ms-cache", "ms-cache v2", "SQL 2005", "SQL 2008", "MD5" ] }, "dictionary": { "title": "Dictionary", "type": "string", "enum": [ "crackstation", "crackstation-human-only", "m3g9tr0n", "words-english" ] }, "rules": { "title": "Rule File", "type": "string", "enum": [ "d3ad0ne", "t0xic" ] }, "hashes": { "title": "Hashes", "type": "string", "description": "Note: Use the file format as required by hashcat" } } } },
        { "toolid": "1cee8439-7f22-457c-84b8-5a8b04414090", "name": "John the Ripper", "version": "1.8.0", "form": [ "algorithm", "dictionary", "rules", { "key": "hashes", "type": "textarea", "placeholder": "Hashes go here!" } ], "schema": { "type": "object", "properties": { "algorithm": { "title": "Algorithm", "type": "string", "enum": [ "NTLM", "NTLMv2", "ms-cache", "ms-cache v2", "SQL 2005", "SQL 2008", "MD5" ] }, "dictionary": { "title": "Dictionary", "type": "string", "enum": [ "crackstation", "crackstation-human-only", "m3g9tr0n", "words-english" ] }, "rules": { "title": "Rule File", "type": "string", "enum": [ "d3ad0ne", "t0xic" ] }, "hashes": { "title": "Hashes", "type": "string", "description": "Note: Use the file format as required by hashcat" } } } }
    ];

    this.query = function() {
        var tmpdata = angular.copy(this.data);
        for (var i = 0; i < tmpdata.length; i++) {
            delete tmpdata[i]["form"];
            delete tmpdata[i]["schema"];
        }
        return tmpdata;
    };

    this.read = function(id) {
        var found = $filter('filter')(this.data, {toolid: id}, true);
        if(found.length) {
            return found[0];
        } else {
            return false;
        }
    };
});

angular.module('cracklord').service('ResourcesDataModel', function ResourcesDataModel($filter) {
    this.data = [
        { "resourceid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1", "status": "running", "hardware": { "gpu": { "1424133520":0.2, "1424144520":0.3 }, "cpu": { "1424133520":0.7, "1424144520":0.9 } }, "tools": { "63ee8045-966f-449e-9839-58e7e0586f3c": { "name": "Hashcat", "version": "1.3.3" }, "8d660ce9-f15d-40a3-a997-a4e8867cb802": { "name": "John the Ripper", "version": "1.7.9" }, "8d660ce9-f15d-328b-a997-39dl10d012ld": { "name": "John the Ripper", "version": "1.8.0" } } },
        { "resourceid": "202fa763-ab6d-4cad-b29d-5fa108766760", "status": "paused", "hardware": { "gpu": { "1424133520":0.2, "1424144520":0.3 }, "cpu": { "1424133520":0.7, "1424144520":0.9 } }, "tools": { "63ee8045-966f-449e-9839-58e7e0586f3c": { "name": "Hashcat", "version": "1.3.3" }, "8d660ce9-f15d-40a3-a997-a4e8867cb802": { "name": "John the Ripper", "version": "1.7.9" }, "8d660ce9-f15d-328b-a997-39dl10d012ld": { "name": "John the Ripper", "version": "1.8.0" } } }
    ];

    this.query = function() {
        var tmpdata = angular.copy(this.data);
        for (var i = 0; i < tmpdata.length; i++) {
            delete tmpdata[i]["hardware"];
            delete tmpdata[i]["tools"];
        }
        return tmpdata;
    };

    this.read = function(id) {
        var found = $filter('filter')(this.data, {resourceid: id}, true);
        if(found.length) {
            return found[0];
        } else {
            return false;
        }
    }

    this.newUUID = function() {
        function s4() {
            return Math.floor((1 + Math.random()) * 0x10000).toString(16).substring(1);
        }
        return s4() + s4() + '-' + s4() + '-' + s4() + '-' + s4() + '-' + s4() + s4() + s4(); 
    };

    this.create = function(data) {
        var newID = this.newUUID();
        this.data[newID] = { "status": "running", "hardware": { "gpu": { "1424133520":0.2, "1424144520":0.3 }, "cpu": { "1424133520":0.7, "1424144520":0.9 } }, "tools": { "63ee8045-966f-449e-9839-58e7e0586f3c": { "name": "Hashcat", "version": "1.3.3" }, "8d660ce9-f15d-40a3-a997-a4e8867cb802": { "name": "John the Ripper", "version": "1.7.9" }, "8d660ce9-f15d-328b-a997-39dl10d012ld": { "name": "John the Ripper", "version": "1.8.0" } } };
        return newID;
    }

    this.update = function(id, data) {
        for(var i = 0; i < this.data.length; i++) {
            if(this.data[i].resourceid === id) {
                this.data[i] = data;
                return this.data[i];;
            }
        }

        return false;
    }

    this.delete = function(id) {
        var match = false;
        for(var i = 0; i < this.data.length; i++) {
            if(this.data[i].jobid = id) {
                match = true;
                this.data.splice(i, 1);
                break;
            }
        }
        return match;    
    }; 
});

angular.module('cracklord').service('JobsDataModel', function JobsDataModel($filter) {
    this.data = [
        {"jobid":"72fd24ca-e529-4b38-b70d-2ad566de7e49", "name":"The Cheerful Shark Logistics Company","status":"running","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423978660,"crackedhashes":5,"totalhashes":800,"progress":0.68, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"786c4f68-1b7f-46e0-b5bd-75090d78b25c", "name":"The Deep Lime Builders Company","status":"paused","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423739455,"crackedhashes":102,"totalhashes":539,"progress":0.17,  "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"af231ea6-5ec4-4bd8-a5f8-5317b69c1b36", "name":"The Little Cow Builders Company","status":"paused","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423521017,"crackedhashes":5,"totalhashes":72,"progress":0.47, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"de83f0e9-f53a-4e43-9285-0fc0e01e1ed7", "name":"Stupid Pigeon Films","status":"paused","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423703738,"crackedhashes":98,"totalhashes":755,"progress":0.85, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"9000a92c-cabf-45aa-97a1-dd739c42e5fc", "name":"Jealous Tiger Web Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1424040300,"crackedhashes":22,"totalhashes":148,"progress":0.71, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"b762b17a-c324-4385-8629-a829e1bc4395", "name":"Beta Gecko Films","status":"created","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423494683,"crackedhashes":0,"totalhashes":352,"progress":0, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"901b0806-4870-43d5-b270-5034f779f55f", "name":"The Alpha Camel Builders Company","status":"created","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423566810,"crackedhashes":0,"totalhashes":698,"progress":0, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"ac61c389-e49d-4d10-8f1d-b54e69690a87", "name":"Beta Tomato Marketing","status":"created","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423498166,"crackedhashes":0,"totalhashes":199,"progress":0, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"bca93fb6-a609-4e66-b70b-2ed84c09ad78", "name":"The Deep Chinchilla Corporation","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423668946,"crackedhashes":54,"totalhashes":398,"progress":0.87, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"3257efbb-44e2-4bd6-b087-8f042e17e5a6", "name":"Rainy Pen Builders","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423675695,"crackedhashes":86,"totalhashes":994,"progress":0.85, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"f3448427-da97-402c-958d-465c48ef0fc8", "name":"Green Phone Web Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423581903,"crackedhashes":1,"totalhashes":10,"progress":0.47, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"f4db813f-5db7-47d1-9d9d-d5071ef135ff", "name":"The Ice-Cold Sheep Web Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423816312,"crackedhashes":14,"totalhashes":74,"progress":0.10, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"c3a393eb-76aa-43af-adbf-e0973a6480ee", "name":"Foggy Meerkat Builders","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423788340,"crackedhashes":71,"totalhashes":924,"progress":0.19, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"1b76c21b-5e02-4b76-ae08-7f9663ae7e6b", "name":"Small Whale Web Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423947518,"crackedhashes":67,"totalhashes":436,"progress":0.76, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"fb4223de-9a65-40b3-b78b-fd549e10e726", "name":"The Freezing Tree Company","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1424000049,"crackedhashes":134,"totalhashes":680,"progress":0.94, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"68ab4678-a9f5-457a-af64-528e7d5810c6", "name":"The Orange Lime Company","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423626046,"crackedhashes":29,"totalhashes":709,"progress":0.04, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"3a3c84d4-ca65-4628-b3cf-cc72cf9594cf", "name":"The Blue Mouse Films Company","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423837255,"crackedhashes":47,"totalhashes":721,"progress":0.31, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"407c642e-fe4c-4b2d-9516-5ddc79e89376", "name":"Sad Orange Web Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1424041731,"crackedhashes":60,"totalhashes":367,"progress":0.92, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"9b197d1a-df60-4c7e-b801-3ed42868d8cb", "name":"Deep Beaver Print Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1424070022,"crackedhashes":49,"totalhashes":760,"progress":0.35, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"ff14abc5-3284-4152-b4e0-1fa2057e3567", "name":"Little Dog Films","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423882378,"crackedhashes":138,"totalhashes":970,"progress":0.30, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"456f83e8-257d-4b78-908d-eee6f632a62c", "name":"Sunny Shark Bank","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423681321,"crackedhashes":10,"totalhashes":140,"progress":0.18, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"b7ee938f-2c67-43f6-b57d-9bad5a6423b0", "name":"Opaque Robot Bank","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423786731,"crackedhashes":50,"totalhashes":939,"progress":0.18, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"126947b7-c00a-4047-b447-d5b4d5134774", "name":"The Black Cherry Bank","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1423617778,"crackedhashes":17,"totalhashes":208,"progress":0.33, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"45194ea0-b60b-430f-9be1-d14d7195620a", "name":"The Cold Duck theatre Company","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"jmmcatee","starttime":1424015948,"crackedhashes":54,"totalhashes":321,"progress":0.82, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"6f32a524-504a-41fb-9a95-7f2598f12566", "name":"The Ice-Cold Sheep Web Design Company","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1424045947,"crackedhashes":121,"totalhashes":809,"progress":0.73, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"aa5f1417-7c3c-440f-947a-d32c96eb3b6a", "name":"White Box Builders","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423540885,"crackedhashes":83,"totalhashes":585,"progress":0.31, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"628c1286-05b5-4643-b9d2-07d53fc0f36e", "name":"The Purple Pen Trading Company","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423873796,"crackedhashes":177,"totalhashes":916,"progress":0.03, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"9ed731cc-4e7a-494d-8ec8-4d6a1c52e530", "name":"Blue Fan Web Design","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423515472,"crackedhashes":64,"totalhashes":812,"progress":0.59, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"1906f26b-13ea-4e88-a58a-d63f307d1018", "name":"The Brown Moose","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1423854430,"crackedhashes":12,"totalhashes":503,"progress":0.92, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
        {"jobid":"1b26fd52-d0d4-4a3a-9dfb-3e122e6eadf1", "name":"Big Zebra Builders","status":"quit","resourceid":"1116814b-7c59-4b5d-87b6-fabaa5f594d1","owner":"emperorcow","starttime":1424054561,"crackedhashes":35,"totalhashes":441,"progress":0.7, "params": { "rules": "d3ad0ne", "dictionary": "m3g9tr0n", "algorithm": "NTLMv2", "hashes": "Administrator:500:6A98EB0FB88A449CBE6FABFD825BCA61:D144986C6122B1B1654BA39932465528:::\nGuest:501:A0E150C75A17008EAAD3B435B51404EE:3D2B4DFAC512B7EF6188248B8E113CB9:::\nfakeuser:1000:24500AFA4E78B1C1944E2DF489A880E4:F47E4045A58ECEBD1F544168E050B1A9:::"}, "toolid": "1116814b-7c59-4b5d-87b6-fabaa5f594d1"},
    ];
   
    this.queueQuery = function() {
        var tmpdata = [];
    }


    this.query = function() {
        var tmpdata = angular.copy(this.data);
        for (var i = 0; i < tmpdata.length; i++) {
            delete tmpdata[i]["params"];
        }
        return tmpdata;
    };

    this.read = function(id) {
        var found = $filter('filter')(this.data, {jobid: id}, true);
        if(found.length) {
            return found[0];
        } else {
            return false;
        }
    };
 
    this.newUUID = function() {
        function s4() {
            return Math.floor((1 + Math.random()) * 0x10000).toString(16).substring(1);
        }
        return s4() + s4() + '-' + s4() + '-' + s4() + '-' + s4() + '-' + s4() + s4() + s4(); 
    };

    this.create = function(data) {
        var newJob = {};
        var newID = this.newUUID();

        newJob.jobid = newID;
        newJob.name = data["name"];
        newJob.status = "created";
        newJob.toolid = data["toolid"];
        newJob.resourceid = "1116814b-7c59-4b5d-87b6-fabaa5f594d1";
        newJob.owner = "emperorcow";
        newJob.starttime = 0;
        newJob.crackedhashes = 0;
        newJob.totalhashes = data['params']['hashes'].split(/\r\n|\r|\n/).length;
        newJob.progress = 0;
        newJob.params = data["params"];

        this.data.push(newJob);
        return newID;
    }

    this.update = function(id, data) {
        for(var i = 0; i < this.data.length; i++) {
            if(this.data[i].jobid === id) {
                this.data[i] = data;
                return this.data[i];;
            }
        }

        return false;
    }

    this.delete = function(id) {
        var match = false;
        for(var i = 0; i < this.data.length; i++) {
            if(this.data[i].jobid = id) {
                match = true;
                this.data.splice(i, 1);
                break;
            }
        }
        return match;        
    };
});