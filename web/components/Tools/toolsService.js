cracklord.factory('ToolsService', ['$resource', function ($resource) {
   return $resource('/api/tools/:id', {id: '@id'}, {
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
}]);