cracklord.controller('ToolsController', function ToolsController($scope, ToolsService) {
	this.loadTools = function() {
		var tools = ToolsService.query(
			//Our success handler
			function(data) { },
			//Our error handler
			function(error) {
				growl.error("An error occured while trying to load tools.");
			}
		);
		$scope.tools = tools;
	}

	this.loadTool = function(toolid) {
		var tool = ToolsService.get({id: toolid}, 
			function(data) {
			}, 
			function(error) {
				growl.error("An error occured while trying to load tool information.");
			}
		);
		return tool;
	};

	this.loadTools();
});