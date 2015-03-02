cracklord.controller('JobsController', function JobsController($scope, JobsService, growl) {
	$scope.listreordered = false;
	$scope.now = Math.floor(Date.now() / 1000);
	$scope.jobactions = {};

	$scope.sortableOptions = {
		handle: '.draghandle',
		axis: 'y',
		update: function(e, ui) {
			$scope.listreordered = true;
		}
	};

	$scope.jobactions.play = function(id, index, name) {
		JobsService.update({jobid: id}, {"action": "resume"}, 
			function(successResult) {
				$scope.jobs[index].status = "created";
				growl.success(name+" resumed.");
			}, 
			function(errorResult) {
				growl.error("Error ("+errorResult.status+") occured.");
			}
		);
	}
	$scope.jobactions.pause = function(id, index, name) {
		JobsService.update({jobid: id}, {"action": "pause"}, 
			function(successResult) {
				$scope.jobs[index].status = "paused";
				growl.success(name+" was paused.");
			}, 
			function(errorResult) {
				growl.error("Error ("+errorResult.status+") occured.");
			}
		);
	}
	$scope.jobactions.stop = function(id, index, name) {
		JobsService.update({jobid: id}, {"action": "stop"}, 
			function(successResult) {
				$scope.jobs[index].status = "quit";
				growl.success(name+" was stopped.");
			}, 
			function(errorResult) {
				growl.error("Error ("+errorResult.status+") occured.");
			}
		);
	}

	$scope.jobactions.delete = function(id, index, name) {
		JobsService.delete({jobid: id}, 
			function(successResult) {
				$scope.jobs.splice(index, 1);
				growl.success(name+" was deleted.");
			}, 
			function(errorResult) {
				growl.error("Error ("+errorResult.status+") occured.");
			}
		);
	}

	$scope.reloadJobs = function() {
		$scope.loadJobs();
		growl.success("Data successfully refreshed.")
	}

	$scope.reorderConfirm = function() {

	}
	$scope.reorderCancel = function() {
		$scope.listreordered = false;
		$scope.loadJobs();
		growl.info("Job reorder was cancelled.")
	}

	$scope.loadJobs = function() {
		var jobs = JobsService.query(
			//Our success handler
			function(data) {
				$scope.listreordered = false;
			},
			//Our error handler
			function(error) {
				growl.error("An error occured while trying to load jobs.");
			}
		);
		$scope.jobs = jobs;
	}

	$scope.loadJobs();
});