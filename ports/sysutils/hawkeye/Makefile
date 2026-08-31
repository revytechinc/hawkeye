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

RUN_DEPENDS=	hawkeye-data>0:sysutils/hawkeye-data

USES=		go:modules
GO_MODULE=	github.com/revytechinc/hawkeye
GO_TARGET=	./cmd/hawkeye

# This port ships the hawkeye binary, man pages, sample JSON config, and
# rc.d script. The diagnostic knowledge corpus is RUN_DEPENDS hawkeye-data
# (https://github.com/revytechinc/hawkeye-data). pkg install hawkeye pulls
# the kit; do not vendor the corpus here. hawkeye-data must not depend on
# hawkeye (no cycle).

USE_RC_SUBR=	hawkeye

# RESCUE is off by default. Thin bastille jails often have /rescue as a
# dangling symlink; do not assume it is a real directory. Staging uses
# STAGEDIR (DESTDIR-safe). Local inference is optional; do not depend on
# an uninstalled llama.cpp port.
OPTIONS_DEFINE=	RESCUE
RESCUE_DESC=	Stage /rescue/hawkeye (into a real /rescue; live dangling symlink is replaced)

PLIST_FILES=	bin/hawkeye \
		etc/cloudbsd/hawkeye/config.json.sample \
		share/man/man8/hawkeye.8.gz \
		share/man/man5/hawkeye.conf.5.gz

.include <bsd.port.options.mk>

.if ${PORT_OPTIONS:MRESCUE}
PLIST_FILES+=	/rescue/hawkeye
PLIST_DIRS+=	/boot/hawkeye
.endif

post-install:
.if ${PORT_OPTIONS:MRESCUE}
	${MKDIR} ${STAGEDIR}/rescue
	${INSTALL_PROGRAM} ${STAGEDIR}${PREFIX}/bin/hawkeye ${STAGEDIR}/rescue/hawkeye
	${MKDIR} ${STAGEDIR}/boot/hawkeye
.endif

.include <bsd.port.mk>
