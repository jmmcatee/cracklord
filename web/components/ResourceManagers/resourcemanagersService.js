cracklord.factory('ResourceManagersService', ['$resource', function ($resource) {
   return $resource('/api/resourcemanagers/:id', {id: '@id'}, {
   	query: {
   		isArray: true,
   		method: 'GET',
   		params: {},
   		transformResponse: function(data) {
   			var results = angular.fromJson(data);
   			return results.resourcemanagers;
   		}	
   	},
   });
}]);

cracklord.service('ResourceManagers', ['ResourceManagersService', '$filter', function(ResourceManagersService, $filter) {
   var managers = {};
   managers.list = [];
   managers.names = {};

   managers.load = function() {
      return ResourceManagersService.query(
         function(data) {
            angular.copy(data, managers.list);
            for(var i = 0; i < data.length; i++) {
               managers.names[data[i].id] = data[i].name;
            }
         }
      );
   }

   managers.idToName = function(id) {
      return managers.names[id];
   }

   managers.get = function(id) {
      return ResourceManagersService.get({id: id});
   }

   return managers;
}]);
