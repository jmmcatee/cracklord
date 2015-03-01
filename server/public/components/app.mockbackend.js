angular.module('cracklord').run(function($httpBackend, JobsDataModel) {
    
    $httpBackend.whenGET('/api/jobs').respond(function(method, url, data) {
        var jobs = JobsDataModel.listJobs();
        var returninfo = { "status": 200, "message": "OK", "jobs": jobs};
        return [200, returninfo, {}];
    });

    $httpBackend.whenGET(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var jobid = url.split('/')[2];
        
        var job = JobsDataModel.readJob(jobid);

        if(job != false) {
            job["status"] = 200;
            job["message"] = "OK";

            return job;
        } else {
            return {"status": 404, "message": "Job "+jobid+" not found."};
        }
    });

    $httpBackend.whenPOST('/api/jobs').respond(function(method, url, data) {
        var params = angular.fromJson(data);
        var jobid = JobsDataModel.createJob(params);
        
        return {"status": 201, "message": "Job "+jobid+" successfully created.", "jobid": jobid};
    });

    $httpBackend.whenPUT(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var params = angular.fromJson(data);
        var jobid = url.split('/')[2];
        var result = JobsDataModel.updateJob(jobid, params);
       
        if(result == true) {
            return {"status": 200, "message": "OK"};
        } else {
            return  { "status": 404, "message": "Job not found" };
        }
    });
    
    $httpBackend.whenDELETE(/\/jobs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/).respond(function(method, url, data) {
        var jobid = url.split('/')[2];
        var result = JobsDataModel.deleteOne(jobid);
        
        if(result == true) {
            return {"status": 200, "message": "OK"};
        } else {
            return  { "status": 404, "message": "Job not found" };
        }
    });    

    $httpBackend.whenGET(/components/).passThrough();
});

