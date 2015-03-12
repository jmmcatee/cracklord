cracklord.directive('jobplaybutton', function (growl) {
	return {
		restrict: 'E',
		replace: true,
		template: '<button aria-label="play" type="button" class="btn"><i class="fa fa-play"></i></button>',
		scope: {
			job: '='			
		},
		controller: function($scope) {
			$scope.doOnClick = function() {
				job.status = 'created';
				job.$update({jobid: job.jobid}, 
					function(successResult) {
						growl.success(job.name+' resumed successfully.');
					}, 
					function(errorResult) {
						growl.error("Error ".errorResult.message);
					}
				);
			}
		},
		link: function($scope, $element, $attrs) {
			if(!isAuthorized([userRoles.standard, userRoles.admin])) {
				$scope.$destroy();
				$scope.remove();
			} else {
				$element.bind('click', function() {
					$scope.doOnClick();
				});
			}
		}
	};
});

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

	$scope.filterJobs = function(status) {
		return (status === 'quit' || status === 'failed');
	}

	$scope.jobactions.update = function(job, status) {
		job.status = status;

		job.$update({jobid: job.jobid},  
			function(successResult) {
				switch(status) {
					case 'created':
						growl.success(job.name+" resumed.");
						break;
					case 'paused':
						growl.success(job.name+" was paused.");
						break;
					case 'quit':
						growl.success(job.name+" was stopped.");
						break;
				}
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

cracklord.controller('JobDetailController', function JobDetailController($scope, JobsService, growl) {
	$scope.loadJobDetail = function(job) {
		job.expanded = true;
		var job = JobsService.get({jobid: job.jobid}, 
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