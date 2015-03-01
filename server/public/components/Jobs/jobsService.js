cracklord.factory('JobsService', function ($resource) {
   return $resource('/api/jobs/:id', {id: '@id'}, {
   	query: {
   		isArray: false,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.jobs;
   		}	
   	}
   });
});