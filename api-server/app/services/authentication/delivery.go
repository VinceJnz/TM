package authentication

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"api-server/v2/models"
)

type MailSender interface {
	SendMail(string, string, string) (bool, error)
}

var ErrRegistrationDelivery = errors.New("failed to send verification email")

func (s Service) SendOTP(ctx context.Context, mailer MailSender, host, name string, user models.User, discardOnFailure bool) (bool, error) {
	token, err := s.Issue(ctx, user.ID, host, name, time.Now().Add(15*time.Minute), "")
	if err != nil || mailer == nil {
		return false, err
	}
	body := fmt.Sprintf("Hi %s,\n\nYour one-time password is: %s\n\nThis code expires in 15 minutes. Enter this code to log in.\n\nIf you didn't request this, please ignore this email.", user.Name, token)
	_, err = deliver(ctx, mailer, user.Email.String, "Your one-time password (OTP)", body)
	if err != nil && discardOnFailure {
		if discardErr := s.DiscardProof(ctx, user.ID, name, token); discardErr != nil {
			log.Print("failed to discard undelivered authentication proof")
		}
	}
	return err == nil, err
}

func (s Service) SendLegacyLoginToken(ctx context.Context, mailer MailSender, host string, user models.User) error {
	token, err := s.Issue(ctx, user.ID, host, "_temp_email_token", time.Now().Add(time.Hour), "")
	if err != nil {
		return err
	}
	if mailer != nil {
		body := fmt.Sprintf("Hi %s,\n\nUse this one-time token to log in: %s\n\nThis token expires in 1 hour.", user.Name, token)
		if _, err := deliver(ctx, mailer, user.Email.String, "Your one-time login token", body); err != nil {
			log.Print("legacy login token email delivery failed")
		}
	}
	return nil
}

func (s Service) SendRegistration(ctx context.Context, mailer MailSender, host string, request RegistrationRequest) error {
	token, err := s.StageRegistration(ctx, host, request)
	if err != nil || mailer == nil {
		return err
	}
	body := fmt.Sprintf("Hi %s,\n\nVerify your email using this code: %s\n\nThis code expires in 24 hours.\n\nIf you didn't register, you can ignore this email.", request.Username, token)
	if _, err := deliver(ctx, mailer, request.Email.String, "Verify your email to complete registration", body); err != nil {
		if err := s.DiscardRegistration(ctx, token); err != nil {
			log.Print("failed to discard pending registration")
		}
		return ErrRegistrationDelivery
	}
	return nil
}

var passwordResetWorkers = make(chan struct{}, 8)

// QueuePasswordReset keeps lookup and delivery outside the response timing.
func (s Service) QueuePasswordReset(ctx context.Context, mailer MailSender, email string) bool {
	select {
	case passwordResetWorkers <- struct{}{}:
	default:
		return false
	}
	go func(ctx context.Context) {
		defer func() { <-passwordResetWorkers }()
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		recipient, code, err := s.CreatePasswordReset(ctx, email)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				log.Print("password reset code creation failed")
			}
			return
		}
		body := "Enter this code in the Forgot password form. It expires in 15 minutes.\n\n" + code +
			"\n\nIf you did not request a reset, you can ignore this message."
		if sent, err := deliver(ctx, mailer, recipient, "Reset your Trip Manager password", body); err != nil || !sent {
			log.Print("password reset email delivery failed")
		}
	}(context.WithoutCancel(ctx))
	return true
}

func deliver(ctx context.Context, mailer MailSender, recipient, subject, body string) (bool, error) {
	if contextual, ok := mailer.(interface {
		SendMailContext(context.Context, string, string, string) (bool, error)
	}); ok {
		return contextual.SendMailContext(ctx, recipient, subject, body)
	}
	return mailer.SendMail(recipient, subject, body)
}
