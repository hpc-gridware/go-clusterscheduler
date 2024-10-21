/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package qsub

import "context"

// Qsub is the interface for submitting jobs using qsub.
type Qsub interface {

	// Submit submits a job with the given options and returns the job ID or an error.
	Submit(ctx context.Context, opts JobOptions) (int64, string, error)

	// SubmitWithNativeSpecification submits a job with the given options and
	// returns the job ID or an error.
	SubmitWithNativeSpecification(ctx context.Context, args []string) (string, error)

	// SubmitSimple submits a simple job script with minimal options.
	SubmitSimple(ctx context.Context, command string, args ...string) (int64, string, error)

	// SubmitSimpleBinary submits a simple executable with minimal options.
	SubmitSimpleBinary(ctx context.Context, command string, args ...string) (int64, string, error)

	// SubmitWithQueue submits a job to a specific queue.
	SubmitWithQueue(ctx context.Context, queue string, opts JobOptions) (int64, string, error)

	// Other simplified methods can be added here.
}
