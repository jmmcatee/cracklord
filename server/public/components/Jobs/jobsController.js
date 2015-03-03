cracklord.controller('JobsController', function JobsController($scope, $modal, JobsService, growl) {
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

	$scope.jobactions.play = function(job) {
		job.status = "created";

		job.$update({jobid: job.jobid},  
			function(successResult) {
				growl.success(job.name+" resumed.");
			}, 
			function(errorResult) {
				growl.error("Error "+errorResult.message);
			}
		);
	}
	$scope.jobactions.pause = function(job) {
		job.status = "paused";

		job.$update({jobid: job.jobid}, 
			function(successResult) {
				growl.success(job.name+" was paused.");
			}, 
			function(errorResult) {
				growl.error("Error "+errorResult.message);
			}
		);
	}
	$scope.jobactions.stop = function(job) {
		job.status = "quit";

		job.$update({jobid: job.jobid}, 
			function(successResult) {
				growl.success(job.name+" was stopped.");
			}, 
			function(errorResult) {
				growl.error("Error "+errorResult.message);
			}
		);
	}

	$scope.jobactions.delete = function(job) {
		var index = $scope.jobs.map(function(el) {
			return el.jobid;
		}).indexOf(job.jobid);

		var name = job.name;
		job.$delete({jobid: job.jobid}, 
			function(successResult) {
				growl.success(name+" was deleted.");
				$scope.jobs.splice(index, 1);
			}, 
			function(errorResult) {
				growl.error("Error "+errorResult.message);
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