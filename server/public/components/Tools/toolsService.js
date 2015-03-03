cracklord.factory('ToolsService', function ($resource) {
   return $resource('/api/tools/:jobid', {jobid: '@jobid'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.tools;
   		}	
   	},
      update: {
         method: 'PUT', 
         transformResponse: function(data) {
            var results = angular.fromJson(data);
            return results.tool;
         }
      }
   });
});