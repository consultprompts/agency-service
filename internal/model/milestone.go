package model

// milestone_index counts COMPLETED milestones: milestone k is complete iff
// milestone_index >= k. The frontend mirrors these values in
// website/src/lib/milestones.ts — keep the two in sync.
const (
	MilestoneNone     = 0 // nothing done yet — waiting for the meeting
	MilestoneMeeting  = 1 // Meeting Completed
	MilestoneMockup   = 2 // Mockup Completed — only ever set together with Approved
	MilestoneApproved = 3 // Design Approved — set when the client approves the mockup
	MilestoneWebsite  = 4 // Website Completed — triggers the payment request
	MilestonePayment  = 5 // Payment Completed — set by payment confirmation only
	MilestoneLive     = 6 // Website is Live
)
