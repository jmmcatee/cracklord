cracklord.controller('JobsController', function JobsController($scope, JobsService){
	$scope.now = Math.floor(Date.now() / 1000);

	$scope.sortableOptions = {
		handle: '.draghandle',
		update: function (e, ui) {
			console.log(e);
			console.log(ui);
		},
		axis: 'y'
	};

	var jobs = JobsService.query(
		//Our success handler
		function(data) {
	
		},
		//Our error handler
		function(error) {
			alert("error");
		}
	);
	$scope.jobs = jobs;
});