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

cracklord.directive('jobsReloadButton', function jobsReloadButton(growl) {
	return {
		restrict: 'E',
		replace: true,
		template: '<button class="btn btn-primary"><i class="fa fa-2x fa-refresh"></i><br> <div class="btnwrd">Refresh</div></button>',
		link: function($scope, $element, $attrs) {
			$element.bind('click', function() {
				$scope.loadJobs();
				growl.success("Data successfully refreshed.");
			});
		}	
	}
});

cracklord.controller('JobDetailController', function JobDetailController($scope, JobsService, growl) {
	$scope.loadJobDetail = function(job) {
		job.expanded = true;
		var job = JobsService.get({id: job.id}, 
			function(data) {
				console.log(data);
			}, 
			function(error) {
				growl.error("An error occured while trying to load tool information.");
			}
		);
	}

});

cracklord.controller('CreateJobController', function CreateJobController($scope, $state, ToolsService, JobsService, growl) {
	$scope.formData = {};
	$scope.formData.params = {};

	$scope.toolChange = function() {
		var toolid = $scope.formData.tool.toolid;
		var tool = ToolsService.get({toolid: toolid}, 
			function(data) {
				$scope.tool = data;
			}, 
			function(error) {
				growl.error("An error occured while trying to load tool information.");
			}
		);
	}

	$scope.processNewJobForm = function() {
		var newjob = new JobsService();

		newjob.toolid = $scope.formData.tool.toolid;
		newjob.name = $scope.formData.name;
		newjob.params = $scope.formData.params;
		
		JobsService.save(newjob, 
			function(data) {
				growl.success("Job successfully added");
				$state.transitionTo('jobs');
			}, 
			function(error) {
				growl.error("An error occured while trying to save the job.");
			}
		);
	}	
});

cracklord.animation('.job-detail', function() {
	return {
		enter: function(element, done) {
			$(element).find('div.slider').slideDown("slow", function() {

			});
		},
		leave: function(element, done) {
			$(element).find('div.slider').slideUp("slow", function() {
				$(element).children('td').hide();
			});
		}
	};	
})