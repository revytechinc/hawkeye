PORTNAME=	hawkeye
DISTVERSION=	0.1.0
CATEGORIES=	sysutils
MASTER_SITES=	LOCAL/revytech/${PORTNAME}/
# Placeholder until the first release tarball is published from
# https://github.com/revytechinc/hawkeye

MAINTAINER=	mark@cloudbsd.org
COMMENT=	FreeBSD diagnostic and operations doctor (MASH)
WWW=		https://github.com/revytechinc/hawkeye

LICENSE=	BSD3CLAUSE
LICENSE_FILE=	${WRKSRC}/LICENSE

USES=		go:modules
GO_MODULE=	github.com/revytechinc/hawkeye
GO_TARGET=	./cmd/hawkeye

# This port ships the hawkeye binary, man pages, sample JSON config, and
# rc.d script only. The diagnostic knowledge corpus is the sibling
# repository/port hawkeye-data (https://github.com/revytechinc/hawkeye-data).
# Do not vendor that corpus here.

USE_RC_SUBR=	hawkeye

PLIST_FILES=	bin/hawkeye \
		etc/cloudbsd/hawkeye/config.json.sample \
		share/man/man8/hawkeye.8.gz \
		share/man/man5/hawkeye.conf.5.gz

.include <bsd.port.mk>
