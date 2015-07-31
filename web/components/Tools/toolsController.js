cracklord.controller('ToolsController', ['$scope', 'ToolsService', function ToolsController($scope, ToolsService) {
	this.loadTools = function() {
		var tools = ToolsService.query(
			//Our success handler
			function(data) { },
			//Our error handler
			function(error) {
				growl.error(error.data.message)
			}
		);
		$scope.tools = tools;
	}

	this.loadTool = function(id) {
		var tool = ToolsService.get({id: id}, 
			function(data) { }, 
			function(error) {
				growl.error(error.data.message)
			}
		);
		return tool;
	};

	this.loadTools();
}]);