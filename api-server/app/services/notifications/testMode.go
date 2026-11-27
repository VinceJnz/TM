package notifications

// SendTestPaymentSuccess sends the notification used by the test-mode redirect.
func SendTestPaymentSuccess(mailer Mailer, email string) error {
	if mailer == nil || email == "" {
		return nil
	}
	_, err := mailer.SendMail(email, "Test Mode - Payment Successful", "Test mode enabled - payment successful for user email:"+email)
	return err
}
