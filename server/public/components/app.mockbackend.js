angular.module('cracklord').run(function($httpBackend, UserDataModel, JobsDataModel, ToolsDataModel, ResourcesDataModel) {
    $httpBackend.whenPOST('/api/login')
    .respond(function(method, url, data) {
       var params = angular.fromJson(data);
       var user = UserDataModel.login(params['username'], params['password']);
        if(user) {
            return [200, {"status": 200, "message": "Login Successful", "token": user.token, "role": user.role}, {}];
        } else {
            return [401, {"status": 401, "message": "Bad username or password."}, {}];
        }
    });
    $httpBackend.whenGET(/\/api\/logout/).passThrough();

    $httpBackend.whenGET('/api/queue').respond(function(method, url, data) {
        var jobs = JobsDataModel.query();
    });

    $httpBackend.whenGET(/\/tools\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var id = url.split('/')[3];
        var tool = ToolsDataModel.read(id);

        if(tool != false) {
            tool["status"] = 200;
            tool["message"] = "OK";
            return [200, tool, {}];
        } else {
            return [404, {"status": 404, "message": "Tool "+id+" not found."}, {}];
        }
    });

    $httpBackend.whenGET('/api/tools').respond(function(method, url, data) {
        var tools = ToolsDataModel.query();
        var returninfo = { "status": 200, "message": "OK", "tools": tools};
        return [200, returninfo, {}];
    });

    $httpBackend.whenGET('/api/resources').respond(function(method, url, data) {
        var resources = ResourcesDataModel.query();
        var returninfo = { "status": 200, "message": "OK", "resources": resources};
        return [200, returninfo, {}];
    });

    $httpBackend.whenGET(/\/resources\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var id = url.split('/')[3];
        var resource = ResourcesDataModel.read(id);

        if(resource != false) {
            resource["status"] = 200;
            resource["message"] = "OK";
            return [200, tool, {}];
        } else {
            return [404, {"status": 404, "message": "Resource "+id+" not found."}, {}];
        }
    });

    $httpBackend.whenGET('/api/jobs').respond(function(method, url, data) {
        var jobs = JobsDataModel.query();
        var returninfo = { "status": 200, "message": "OK", "jobs": jobs};
        return [200, returninfo, {}];
    });

    $httpBackend.whenGET(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var jobid = url.split('/')[3];
        
        var job = JobsDataModel.read(jobid);

        if(job != false) {
            job["status"] = 200;
            job["message"] = "OK";

            return [200, job, {}];
        } else {
            return [404, {"status": 404, "message": "Job "+jobid+" not found."}, {}];
        }
    });

    $httpBackend.whenPOST('/api/jobs').respond(function(method, url, data) {
        var params = angular.fromJson(data);
        var jobid = JobsDataModel.create(params);
        
        return [201, {"status": 201, "message": "Job "+jobid+" successfully created.", "jobid": jobid}, {}];
    });

    $httpBackend.whenPUT(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var params = angular.fromJson(data);
        var jobid = url.split('/')[3];
        var result = JobsDataModel.update(jobid, params);
       
        if(!result) {
            return [404, { "status": 404, "message": "Job not found" }, {}];
        } else {
            return [200, {"status": 200, "message": "OK", "job": result}, {}];
        }
    });
    
    $httpBackend.whenDELETE(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var jobid = url.split('/')[3];
        var result = JobsDataModel.delete(jobid);
        
        if(result == true) {
            return [200, {"status": 200, "message": "OK"}, {}];
        } else {
            return [404, {"status": 404, "message": "Job not found"}, {}];
        }
    });    

    $httpBackend.whenGET(/components/).passThrough();
});

