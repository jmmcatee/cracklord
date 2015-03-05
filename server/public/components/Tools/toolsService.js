cracklord.factory('ToolsService', function ($resource) {
   return $resource('/api/tools/:toolid', {toolid: '@toolid'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.tools;
   		}	
   	},
   });
});