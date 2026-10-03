package notideus

type EmailsService struct{ client *Client }

type WhatsAppService struct {
	client   *Client
	Messages *WhatsAppMessagesService
}

type WhatsAppMessagesService struct{ client *Client }

type ContactsService struct{ client *Client }

type UnsubscribeService struct{ client *Client }

type PlansService struct{ client *Client }
