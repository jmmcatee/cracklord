cracklord.controller('ResourcesController', ['$scope', 'ResourceList', 'ResourceManagers', function ResourceController($scope, ResourceList, ResourceManagers) {
	$scope.managers = ResourceManagers.list; 
	ResourceManagers.load();

	$scope.resources = ResourceList.list;
	ResourceList.load();
	$scope.loadServers = ResourceList.update(); 
}]);

cracklord.controller('ConnectResourceController', ['$scope', '$state', 'ResourceService', 'ResourceList', 'ResourceManagers', 'growl', '$timeout', '$stateParams', function CreateJobController($scope, $state, ResourceService, ResourceList, ResourceManagers, growl, $timeout, $stateParams) {
	$scope.formData = {};
	$scope.displayWait = false;
	ResourceManagers.get($stateParams.manager).$promise.then(
		function(data) {
			$scope.manager = data.resourcemanager
		}, 
		function(error) {
			growl.error("Unable to load resource manager parameters: "+error.message);
		}
	)

	$scope.processResourceConnectForm = function() {
		$scope.displayWait = true;

		var newresource = new ResourceService();

		newresource.manager = $scope.manager.id;
		newresource.params = $scope.formData;
		
		ResourceService.save(newresource).$promise.then(
			function(data) {
				$timeout(function() {
					ResourceList.update();
					growl.success("Successfully submitted resource connection request.");
					$state.transitionTo('resources');
				}, 1000)
			}, 
			function(error) {
				$scope.displayWait = false;
				switch (error.status) {
					case 400: growl.error("You sent bad data, check your input and if it's correct get in touch with us on github"); break;
					case 403: growl.error("You're not allowed to do that..."); break;
					default: growl.error(error.message); break;
				}
			}
		);
	}	
}]);
