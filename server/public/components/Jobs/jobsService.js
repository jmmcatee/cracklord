cracklord.factory('JobsService', function ($resource) {
   return $resource('/api/jobs/:jobid', {jobid: '@jobid'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.jobs;
   		}	
   	},
      update: {
         method: 'PUT', 
         transformResponse: function(data) {
            var results = angular.fromJson(data);
            return results.job;
         }
      }
   });
});
